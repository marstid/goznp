package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/marstid/goznp/pkg/zcl"
	"github.com/marstid/goznp/pkg/znp"
)

// Groups Cluster (0x0004)

// AddToGroup adds a device endpoint to a group.
// groupID is the 16-bit group address (1-65535).
// groupName is optional (can be empty string).
func (a *Adapter) AddToGroup(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16, groupName string) error {
	// AddGroup payload: groupId (2 bytes LE) + groupName (string with length prefix)
	nameBytes := []byte(groupName)
	if len(nameBytes) > 16 {
		nameBytes = nameBytes[:16] // ZCL limits group names to 16 chars
	}

	payload := make([]byte, 3+len(nameBytes))
	payload[0] = byte(groupID & 0xFF)
	payload[1] = byte(groupID >> 8)
	payload[2] = byte(len(nameBytes))
	copy(payload[3:], nameBytes)

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterGroups, zcl.CmdGroupsAdd, payload)
}

// RemoveFromGroup removes a device endpoint from a group.
func (a *Adapter) RemoveFromGroup(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16) error {
	// RemoveGroup payload: groupId (2 bytes LE)
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterGroups, zcl.CmdGroupsRemove, payload)
}

// RemoveFromAllGroups removes a device endpoint from all groups.
func (a *Adapter) RemoveFromAllGroups(ctx context.Context, nwkAddr uint16, endpoint uint8) error {
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterGroups, zcl.CmdGroupsRemoveAll, nil)
}

// GroupMembership contains the groups a device belongs to.
type GroupMembership struct {
	Capacity uint8    // Remaining capacity for group memberships (0xFF = unknown)
	Groups   []uint16 // List of group IDs the device belongs to
}

// GetGroupMembership queries which groups a device endpoint belongs to.
func (a *Adapter) GetGroupMembership(ctx context.Context, nwkAddr uint16, endpoint uint8) (*GroupMembership, error) {
	// GetGroupMembership payload: groupCount (1 byte) + groupList (empty = query all)
	payload := []byte{0x00} // Query all groups

	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Build and send request
	seqNum := a.nextTransactionID()
	frame := zcl.BuildClusterCommand(seqNum, zcl.CmdGroupsGetMembership, payload)

	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(zcl.ClusterGroups), frame)
	if err != nil {
		return nil, fmt.Errorf("get group membership failed: %w", err)
	}

	// Parse response (command 0x02)
	if respFrame.CommandID != zcl.CmdGroupsGetMembershipResponse {
		return nil, fmt.Errorf("unexpected response command: 0x%02X", respFrame.CommandID)
	}

	if len(respFrame.Payload) < 2 {
		return nil, fmt.Errorf("response too short")
	}

	result := &GroupMembership{
		Capacity: respFrame.Payload[0],
		Groups:   make([]uint16, 0),
	}

	groupCount := respFrame.Payload[1]
	offset := 2
	for i := uint8(0); i < groupCount && offset+1 < len(respFrame.Payload); i++ {
		groupID := uint16(respFrame.Payload[offset]) | uint16(respFrame.Payload[offset+1])<<8
		result.Groups = append(result.Groups, groupID)
		offset += 2
	}

	return result, nil
}

// SendGroupCommand sends a cluster command to all devices in a group.
// This uses group addressing (multicast) so the command is received by all
// group members simultaneously. groupID is the 16-bit group address.
// clusterID is the cluster to send the command on, commandID is the
// cluster-specific command, and payload is the command payload.
//
// Example: Turn on all lights in group 1:
//
//	err := adapter.SendGroupCommand(ctx, 1, zcl.ClusterOnOff, zcl.CmdOnOffOn, nil)
func (a *Adapter) SendGroupCommand(ctx context.Context, groupID uint16, clusterID zcl.ClusterID, commandID uint8, payload []byte) error {
	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return ErrNotOpen
	}
	znpClient := a.znp
	a.mu.Unlock()

	// Build cluster command frame
	seqNum := a.nextTransactionID()
	frame := zcl.BuildClusterCommand(seqNum, commandID, payload)

	// Build AF data request with group addressing
	// For group addressing, we use the group ID as the destination address
	// and broadcast endpoint (0xFF) as the destination endpoint
	req := znp.DataRequest{
		DstAddr:     groupID, // Group ID as destination
		DstEndpoint: 0xFF,    // Broadcast endpoint for group commands
		SrcEndpoint: CoordinatorEndpoint,
		ClusterID:   uint16(clusterID),
		TransID:     seqNum,
		Options:     znp.AfOptionNone, // No ACK for group commands
		Radius:      30,
		Data:        frame.ToBytes(),
	}

	// Send request
	status, err := znpClient.AfDataRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("group data request failed: %w", err)
	}

	if status != 0 {
		return fmt.Errorf("group data request returned status 0x%02X", status)
	}

	// Wait for data confirm (no response expected from devices)
	confirm, err := znpClient.WaitForDataConfirm(ctx, seqNum, 5*time.Second)
	if err != nil {
		return fmt.Errorf("waiting for data confirm: %w", err)
	}

	if confirm.Status != 0 {
		return fmt.Errorf("data confirm returned status 0x%02X", confirm.Status)
	}

	return nil
}

// GroupTurnOn sends On command to all devices in a group.
func (a *Adapter) GroupTurnOn(ctx context.Context, groupID uint16) error {
	return a.SendGroupCommand(ctx, groupID, zcl.ClusterOnOff, zcl.CmdOnOffOn, nil)
}

// GroupTurnOff sends Off command to all devices in a group.
func (a *Adapter) GroupTurnOff(ctx context.Context, groupID uint16) error {
	return a.SendGroupCommand(ctx, groupID, zcl.ClusterOnOff, zcl.CmdOnOffOff, nil)
}

// GroupToggle sends Toggle command to all devices in a group.
func (a *Adapter) GroupToggle(ctx context.Context, groupID uint16) error {
	return a.SendGroupCommand(ctx, groupID, zcl.ClusterOnOff, zcl.CmdOnOffToggle, nil)
}

// GroupSetBrightness sets the brightness level on all dimmable devices in a group.
// level is 0-254 (0=off, 254=full brightness).
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
func (a *Adapter) GroupSetBrightness(ctx context.Context, groupID uint16, level uint8, transitionTime uint16) error {
	payload := []byte{
		level,
		byte(transitionTime & 0xFF),
		byte(transitionTime >> 8),
	}
	return a.SendGroupCommand(ctx, groupID, zcl.ClusterLevelControl, zcl.CmdLevelMoveToLevelWithOnOff, payload)
}

// GroupRecallScene recalls a scene on all devices in a group.
// sceneID is the scene to recall (0-255).
func (a *Adapter) GroupRecallScene(ctx context.Context, groupID uint16, sceneID uint8) error {
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
		sceneID,
	}
	return a.SendGroupCommand(ctx, groupID, zcl.ClusterScenes, zcl.CmdScenesRecall, payload)
}

// Scenes Cluster (0x0005)

// StoreScene stores the current device state as a scene.
// The device saves its current attribute values (brightness, color, etc.) to the scene.
// groupID must be a valid group the device belongs to (or 0x0000 for global scenes).
// sceneID is 0-255.
func (a *Adapter) StoreScene(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16, sceneID uint8) error {
	// StoreScene payload: groupId (2 bytes LE) + sceneId (1 byte)
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
		sceneID,
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterScenes, zcl.CmdScenesStore, payload)
}

// AddScene adds a scene with explicit parameters.
// This is an advanced command that allows specifying scene details explicitly
// rather than capturing current device state like StoreScene does.
// groupID must be a valid group the device belongs to (or 0x0000 for global scenes).
// sceneID is 0-255.
// transitionTime is in tenths of a second (e.g., 10 = 1 second).
// sceneName is optional (max 16 characters).
// Note: This only creates the scene metadata. To capture device state, use StoreScene instead.
func (a *Adapter) AddScene(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16, sceneID uint8, transitionTime uint16, sceneName string) error {
	// AddScene payload: groupId (2 bytes LE) + sceneId (1 byte) + transitionTime (2 bytes LE) + sceneName (string with length prefix)
	nameBytes := []byte(sceneName)
	if len(nameBytes) > 16 {
		nameBytes = nameBytes[:16] // ZCL limits scene names to 16 chars
	}

	payload := make([]byte, 6+len(nameBytes))
	payload[0] = byte(groupID & 0xFF)
	payload[1] = byte(groupID >> 8)
	payload[2] = sceneID
	payload[3] = byte(transitionTime & 0xFF)
	payload[4] = byte(transitionTime >> 8)
	payload[5] = byte(len(nameBytes))
	copy(payload[6:], nameBytes)

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterScenes, zcl.CmdScenesAdd, payload)
}

// RecallScene activates a previously stored scene.
// The device transitions to the saved attribute values.
// transitionTime is optional (0xFFFF = use scene's stored transition time).
func (a *Adapter) RecallScene(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16, sceneID uint8, transitionTime *uint16) error {
	// RecallScene payload: groupId (2 bytes LE) + sceneId (1 byte) + [transitionTime (2 bytes LE)]
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
		sceneID,
	}

	// Add optional transition time
	if transitionTime != nil {
		payload = append(payload, byte(*transitionTime&0xFF), byte(*transitionTime>>8))
	}

	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterScenes, zcl.CmdScenesRecall, payload)
}

// RemoveScene removes a specific scene from the device.
func (a *Adapter) RemoveScene(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16, sceneID uint8) error {
	// RemoveScene payload: groupId (2 bytes LE) + sceneId (1 byte)
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
		sceneID,
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterScenes, zcl.CmdScenesRemove, payload)
}

// RemoveAllScenes removes all scenes for a group from the device.
func (a *Adapter) RemoveAllScenes(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16) error {
	// RemoveAllScenes payload: groupId (2 bytes LE)
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
	}
	return a.SendClusterCommand(ctx, nwkAddr, endpoint, zcl.ClusterScenes, zcl.CmdScenesRemoveAll, payload)
}

// SceneMembership contains the scenes stored on a device for a group.
type SceneMembership struct {
	Status   zcl.Status // ZCL status (0 = success)
	Capacity uint8      // Remaining capacity for scenes (0xFF = unknown)
	GroupID  uint16     // The group these scenes belong to
	Scenes   []uint8    // List of scene IDs
}

// GetSceneMembership queries which scenes are stored on a device for a group.
func (a *Adapter) GetSceneMembership(ctx context.Context, nwkAddr uint16, endpoint uint8, groupID uint16) (*SceneMembership, error) {
	// GetSceneMembership payload: groupId (2 bytes LE)
	payload := []byte{
		byte(groupID & 0xFF),
		byte(groupID >> 8),
	}

	a.mu.Lock()
	if !a.isOpen {
		a.mu.Unlock()
		return nil, ErrNotOpen
	}
	a.mu.Unlock()

	// Build and send request
	seqNum := a.nextTransactionID()
	frame := zcl.BuildClusterCommand(seqNum, zcl.CmdScenesGetMembership, payload)

	respFrame, err := a.sendZclRequest(ctx, nwkAddr, endpoint, uint16(zcl.ClusterScenes), frame)
	if err != nil {
		return nil, fmt.Errorf("get scene membership failed: %w", err)
	}

	// Parse response (command 0x06)
	if respFrame.CommandID != zcl.CmdScenesGetMembershipResponse {
		return nil, fmt.Errorf("unexpected response command: 0x%02X", respFrame.CommandID)
	}

	if len(respFrame.Payload) < 4 {
		return nil, fmt.Errorf("response too short")
	}

	result := &SceneMembership{
		Status:   zcl.Status(respFrame.Payload[0]),
		Capacity: respFrame.Payload[1],
		GroupID:  uint16(respFrame.Payload[2]) | uint16(respFrame.Payload[3])<<8,
		Scenes:   make([]uint8, 0),
	}

	// Only parse scene list if status is success
	if result.Status == zcl.StatusSuccess && len(respFrame.Payload) > 4 {
		sceneCount := respFrame.Payload[4]
		for i := uint8(0); i < sceneCount && int(5+i) < len(respFrame.Payload); i++ {
			result.Scenes = append(result.Scenes, respFrame.Payload[5+i])
		}
	}

	return result, nil
}
