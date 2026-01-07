package ota

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
	"time"
)

// Server handles OTA upgrade requests from Zigbee devices.
// It serves firmware images and tracks upgrade progress.
type Server struct {
	mu           sync.RWMutex
	images       map[string]*Image           // Indexed by manufacturer:imageType.
	progress     map[string]*UpgradeProgress // Indexed by IEEE address.
	imagesDir    string                      // Directory containing OTA files.
	maxBlockSize uint8                       // Maximum block size to serve.
	serverAddr   [8]byte                     // IEEE address of this OTA server.

	// Callbacks for progress updates.
	OnProgress func(ProgressUpdate)
	OnComplete func(device string, fileVersion uint32)
	OnError    func(device string, err error)
}

// ServerConfig holds configuration for the OTA server.
type ServerConfig struct {
	// ImagesDir is the directory containing OTA firmware files.
	ImagesDir string

	// MaxBlockSize is the maximum block size to serve (default: 64).
	MaxBlockSize uint8

	// MinBlockDelay is the minimum delay between block requests (default: 50ms).
	MinBlockDelay uint16

	// MaxBlockDelay is the maximum delay between block requests (default: 500ms).
	MaxBlockDelay uint16

	// ServerAddr is the IEEE address of the OTA server (defaults to coordinator address).
	ServerAddr [8]byte

	// Progress callbacks.
	OnProgress func(ProgressUpdate)
	OnComplete func(device string, fileVersion uint32)
	OnError    func(device string, err error)
}

// NewServer creates a new OTA server with the given configuration.
func NewServer(config ServerConfig) (*Server, error) {
	if config.ImagesDir == "" {
		return nil, fmt.Errorf("ota: images directory is required")
	}

	if config.MaxBlockSize == 0 {
		config.MaxBlockSize = 64
	}
	if config.MinBlockDelay == 0 {
		config.MinBlockDelay = 50
	}
	if config.MaxBlockDelay == 0 {
		config.MaxBlockDelay = 500
	}

	s := &Server{
		images:       make(map[string]*Image),
		progress:     make(map[string]*UpgradeProgress),
		imagesDir:    config.ImagesDir,
		maxBlockSize: config.MaxBlockSize,
		serverAddr:   config.ServerAddr,
		OnProgress:   config.OnProgress,
		OnComplete:   config.OnComplete,
		OnError:      config.OnError,
	}

	// Load images from directory.
	if err := s.loadImages(); err != nil {
		return nil, fmt.Errorf("ota: failed to load images: %w", err)
	}

	return s, nil
}

// loadImages scans the images directory and loads all OTA files.
func (s *Server) loadImages() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.images = make(map[string]*Image)

	err := filepath.WalkDir(s.imagesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories.
		if d.IsDir() {
			return nil
		}

		// Skip hidden files and common non-OTA files.
		name := d.Name()
		if name[0] == '.' {
			return nil
		}
		if filepath.Ext(name) != ".ota" && filepath.Ext(name) != ".zigbee" && filepath.Ext(name) != ".ota2" {
			return nil
		}

		// Parse the OTA file.
		img, err := ParseFile(path)
		if err != nil {
			// Log error but continue loading other files.
			fmt.Printf("ota: failed to parse %s: %v\n", path, err)
			return nil
		}

		// Index the image.
		key := imageKey(img.ManufacturerCode, img.ImageType, img.FileVersion)
		s.images[key] = img

		return nil
	})

	return err
}

// imageKey creates a lookup key for an OTA image.
func imageKey(manufacturer uint16, imageType uint16, fileVersion uint32) string {
	return fmt.Sprintf("%04X:%04X:%08X", manufacturer, imageType, fileVersion)
}

// ImageFile returns the image type for lookup purposes.
func (i *Image) ImageFile() uint16 {
	return i.ImageType
}

// AddImage adds an OTA image to the server's image database.
func (s *Server) AddImage(img *Image) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := imageKey(img.ManufacturerCode, img.ImageType, img.FileVersion)
	img.Path = "" // Clear path if loaded from bytes.
	s.images[key] = img
}

// RemoveImage removes an OTA image from the server's database.
func (s *Server) RemoveImage(manufacturer uint16, imageType uint16, fileVersion uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := imageKey(manufacturer, imageType, fileVersion)
	delete(s.images, key)
}

// GetImage retrieves an OTA image by manufacturer and image type.
// Returns the latest version if fileVersion is 0, or specific version otherwise.
func (s *Server) GetImage(manufacturer uint16, imageType uint16, fileVersion uint32) (*Image, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if fileVersion != 0 {
		// Looking for specific version.
		key := imageKey(manufacturer, imageType, fileVersion)
		img, ok := s.images[key]
		if !ok {
			return nil, fmt.Errorf("ota: image not found for manufacturer 0x%04X, image type 0x%04X, version %d",
				manufacturer, imageType, fileVersion)
		}
		return img, nil
	}

	// Find latest version.
	var latest *Image
	var latestVersion uint32

	for _, img := range s.images {
		if img.ManufacturerCode == manufacturer && img.ImageType == imageType {
			if img.FileVersion > latestVersion {
				latest = img
				latestVersion = img.FileVersion
			}
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("ota: no image found for manufacturer 0x%04X, image type 0x%04X",
			manufacturer, imageType)
	}

	return latest, nil
}

// ListImages returns all images available in the server.
func (s *Server) ListImages() []*Image {
	s.mu.RLock()
	defer s.mu.RUnlock()

	images := make([]*Image, 0, len(s.images))
	for _, img := range s.images {
		images = append(images, img)
	}
	return images
}

// QueryImage handles a Query Image command from a device.
// Returns the appropriate Query Image Response.
func (s *Server) QueryImage(_ context.Context, req QueryImageCommand, _ string) (*QueryImageResponse, error) {
	// Determine which version to provide.
	fileVersion := req.FileVersion
	if req.FieldControl&0x01 == 0 {
		// Current version not provided, use latest.
		fileVersion = 0
	}

	// Find matching image.
	img, err := s.GetImage(req.ManufacturerCode, req.ImageType, fileVersion)
	if err != nil {
		// No image available.
		return &QueryImageResponse{
			Status:           uint8(StatusNoImageAvailable),
			ManufacturerCode: req.ManufacturerCode,
			ImageType:        req.ImageType,
			ServerAddress:    s.serverAddr,
		}, nil
	}

	// Check compatibility.
	if !img.IsCompatible(req.ManufacturerCode, req.ImageType, req.FileVersion) {
		return &QueryImageResponse{
			Status:           uint8(StatusNoImageAvailable),
			ManufacturerCode: req.ManufacturerCode,
			ImageType:        req.ImageType,
			ServerAddress:    s.serverAddr,
		}, nil
	}

	// Return available image info.
	return &QueryImageResponse{
		Status:            uint8(StatusSuccess),
		ManufacturerCode:  img.ManufacturerCode,
		ImageType:         img.ImageType,
		FileVersion:       img.FileVersion,
		ImageSize:         img.ImageSize,
		ServerAddress:     s.serverAddr,
		MinimumBlockDelay: 0, // Will be set by adapter.
		MaximumBlockDelay: 0, // Will be set by adapter.
	}, nil
}

// ImageBlock handles an Image Block request from a device.
// Returns the appropriate Image Block Response.
func (s *Server) ImageBlock(_ context.Context, req ImageBlockRequest, deviceAddr string) (*ImageBlockResponse, error) {
	// Find the image.
	img, err := s.GetImage(req.ManufacturerCode, req.ImageType, req.FileVersion)
	if err != nil {
		return nil, fmt.Errorf("ota: image not found: %w", err)
	}

	// Validate offset.
	if req.FileOffset >= img.ImageSize {
		return nil, fmt.Errorf("ota: invalid offset %d, image size is %d", req.FileOffset, img.ImageSize)
	}

	// Calculate data size.
	dataSize := req.MaximumDataSize
	if dataSize > s.maxBlockSize {
		dataSize = s.maxBlockSize
	}
	if int(req.FileOffset)+int(dataSize) > len(img.ImageData) {
		dataSize = uint8(len(img.ImageData) - int(req.FileOffset))
	}

	// Extract data block.
	dataOffset := req.FileOffset
	dataEnd := dataOffset + uint32(dataSize)
	if int(dataEnd) > len(img.ImageData) {
		dataEnd = uint32(len(img.ImageData))
	}
	data := make([]byte, dataEnd-dataOffset)
	copy(data, img.ImageData[dataOffset:dataEnd])

	// Update progress tracking.
	s.updateProgress(deviceAddr, req.FileVersion, dataOffset, img.ImageSize)

	// Return block response.
	return &ImageBlockResponse{
		Status:           uint8(StatusSuccess),
		ManufacturerCode: img.ManufacturerCode,
		ImageType:        img.ImageType,
		FileVersion:      req.FileVersion,
		FileOffset:       req.FileOffset,
		DataSize:         dataSize,
		Data:             data,
	}, nil
}

// UpgradeEnd handles an Upgrade End request from a device.
// Returns the appropriate Upgrade End Response.
func (s *Server) UpgradeEnd(_ context.Context, req UpgradeEndRequest, deviceAddr string) (*UpgradeEndResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Determine applicability.
	var applicability uint8
	if req.Status == uint8(StatusSuccess) {
		_, err := s.GetImage(req.ManufacturerCode, req.ImageType, req.FileVersion)
		if err == nil {
			// We'd need to track previous version to determine this accurately.
			// For now, assume upgrade (most common case).
			applicability = 0x03 // Upgraded.
		} else {
			applicability = 0x01 // Downgraded (error case).
		}

		// Notify completion callback.
		if s.OnComplete != nil {
			s.OnComplete(deviceAddr, req.FileVersion)
		}
	} else {
		// Device reported failure.
		applicability = 0x01 // Not applicable.

		// Notify error callback.
		if s.OnError != nil {
			s.OnError(deviceAddr, fmt.Errorf("upgrade failed with status 0x%02X", req.Status))
		}
	}

	// Clean up progress tracking.
	delete(s.progress, deviceAddr)

	return &UpgradeEndResponse{
		ManufacturerCode: req.ManufacturerCode,
		ImageType:        req.ImageType,
		FileVersion:      req.FileVersion,
		Applicability:    applicability,
	}, nil
}

// updateProgress updates the progress tracking for a device.
func (s *Server) updateProgress(deviceAddr string, fileVersion uint32, offset uint32, totalSize uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prog, ok := s.progress[deviceAddr]
	if !ok {
		prog = &UpgradeProgress{
			IEEEAddress:  deviceAddr,
			FileVersion:  fileVersion,
			TotalSize:    totalSize,
			ProgressChan: make(chan ProgressUpdate, 10),
		}
		s.progress[deviceAddr] = prog
	} else {
		prog.FileVersion = fileVersion
		prog.TotalSize = totalSize
	}

	prog.DownloadedSize = offset
	prog.Percentage = float64(offset) / float64(totalSize) * 100.0

	// Send progress update if callback registered.
	if s.OnProgress != nil {
		update := ProgressUpdate{
			Device:      deviceAddr,
			FileVersion: fileVersion,
			Offset:      offset,
			TotalSize:   totalSize,
			Percentage:  prog.Percentage,
			Status:      "downloading",
		}
		s.OnProgress(update)
	}

	// Check for completion.
	if offset >= totalSize {
		prog.Percentage = 100.0
		if s.OnProgress != nil {
			update := ProgressUpdate{
				Device:      deviceAddr,
				FileVersion: fileVersion,
				Offset:      totalSize,
				TotalSize:   totalSize,
				Percentage:  100.0,
				Status:      "complete",
			}
			s.OnProgress(update)
		}
	}
}

// GetProgress returns the current progress for a device.
func (s *Server) GetProgress(deviceAddr string) (*UpgradeProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prog, ok := s.progress[deviceAddr]
	if !ok {
		return nil, fmt.Errorf("ota: no active upgrade for device %s", deviceAddr)
	}
	return prog, nil
}

// MonitorProgress starts monitoring progress for a device via a channel.
func (s *Server) MonitorProgress(ctx context.Context, deviceAddr string) (<-chan ProgressUpdate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.progress[deviceAddr]
	if !ok {
		return nil, fmt.Errorf("ota: no active upgrade for device %s", deviceAddr)
	}

	// Create a buffered channel for updates.
	updates := make(chan ProgressUpdate, 100)

	// Start goroutine to forward progress updates.
	go func() {
		defer close(updates)

		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		lastPercentage := float64(-1)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.mu.RLock()
				if currentProg, found := s.progress[deviceAddr]; found {
					// Send update if percentage changed.
					if currentProg.Percentage != lastPercentage {
						updates <- ProgressUpdate{
							Device:      deviceAddr,
							FileVersion: currentProg.FileVersion,
							Offset:      currentProg.DownloadedSize,
							TotalSize:   currentProg.TotalSize,
							Percentage:  currentProg.Percentage,
							Status:      "downloading",
						}
						lastPercentage = currentProg.Percentage
					}
				} else {
					// Device no longer has active upgrade.
					return
				}
				s.mu.RUnlock()
			}
		}
	}()

	return updates, nil
}

// IsActiveCheck checks if a device has an active OTA upgrade in progress.
func (s *Server) IsActiveCheck(deviceAddr string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.progress[deviceAddr]
	return ok
}

// SetServerAddr sets the IEEE address of the OTA server.
// This is typically the coordinator's IEEE address.
func (s *Server) SetServerAddr(addr [8]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.serverAddr = addr
}

// GetServerAddr returns the IEEE address of the OTA server.
func (s *Server) GetServerAddr() [8]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.serverAddr
}
