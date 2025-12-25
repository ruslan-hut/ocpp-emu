package v201

import (
	"fmt"
	"sync"
)

// Mutability represents whether a variable can be modified
type Mutability string

const (
	MutabilityReadOnly  Mutability = "ReadOnly"
	MutabilityWriteOnly Mutability = "WriteOnly"
	MutabilityReadWrite Mutability = "ReadWrite"
)

// DataType represents the data type of a variable
type DataType string

const (
	DataTypeString       DataType = "string"
	DataTypeDecimal      DataType = "decimal"
	DataTypeInteger      DataType = "integer"
	DataTypeDateTime     DataType = "dateTime"
	DataTypeBoolean      DataType = "boolean"
	DataTypeOptionList   DataType = "OptionList"
	DataTypeSequenceList DataType = "SequenceList"
	DataTypeMemberList   DataType = "MemberList"
)

// VariableCharacteristics describes the characteristics of a variable
type VariableCharacteristics struct {
	DataType        DataType `json:"dataType"`
	SupportsMonitor bool     `json:"supportsMonitoring"`
	Unit            string   `json:"unit,omitempty"`
	MinLimit        *float64 `json:"minLimit,omitempty"`
	MaxLimit        *float64 `json:"maxLimit,omitempty"`
	ValuesList      string   `json:"valuesList,omitempty"` // Comma-separated list of allowed values
}

// VariableAttribute represents a single attribute of a variable
type VariableAttribute struct {
	Type       AttributeType `json:"type"`
	Value      string        `json:"value"`
	Mutability Mutability    `json:"mutability"`
	Persistent bool          `json:"persistent"`
	Constant   bool          `json:"constant"`
}

// VariableInstance represents a variable within a component
type VariableInstance struct {
	Name            string                  `json:"name"`
	Instance        string                  `json:"instance,omitempty"`
	Characteristics VariableCharacteristics `json:"characteristics"`
	Attributes      map[AttributeType]*VariableAttribute
	mu              sync.RWMutex
}

// ComponentInstance represents a component in the device model
type ComponentInstance struct {
	Name      string                       `json:"name"`
	Instance  string                       `json:"instance,omitempty"`
	EVSE      *EVSE                        `json:"evse,omitempty"`
	Variables map[string]*VariableInstance // key: variableName or variableName:instance
	mu        sync.RWMutex
}

// DeviceModel represents the OCPP 2.0.1 device model
type DeviceModel struct {
	Components map[string]*ComponentInstance // key: componentName or componentName:instance
	mu         sync.RWMutex
}

// NewDeviceModel creates a new device model with standard components
func NewDeviceModel() *DeviceModel {
	dm := &DeviceModel{
		Components: make(map[string]*ComponentInstance),
	}
	dm.initializeStandardComponents()
	return dm
}

// getComponentKey generates a unique key for a component
func getComponentKey(name, instance string) string {
	if instance != "" {
		return fmt.Sprintf("%s:%s", name, instance)
	}
	return name
}

// getVariableKey generates a unique key for a variable
func getVariableKey(name, instance string) string {
	if instance != "" {
		return fmt.Sprintf("%s:%s", name, instance)
	}
	return name
}

// GetComponent retrieves a component by name and optional instance
func (dm *DeviceModel) GetComponent(name, instance string) *ComponentInstance {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	key := getComponentKey(name, instance)
	return dm.Components[key]
}

// AddComponent adds a new component to the device model
func (dm *DeviceModel) AddComponent(name, instance string, evse *EVSE) *ComponentInstance {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	key := getComponentKey(name, instance)
	comp := &ComponentInstance{
		Name:      name,
		Instance:  instance,
		EVSE:      evse,
		Variables: make(map[string]*VariableInstance),
	}
	dm.Components[key] = comp
	return comp
}

// GetVariable retrieves a variable from a component
func (dm *DeviceModel) GetVariable(componentName, componentInstance, variableName, variableInstance string, attrType AttributeType) (string, GetVariableStatusType) {
	comp := dm.GetComponent(componentName, componentInstance)
	if comp == nil {
		return "", GetVariableStatusUnknownComponent
	}

	comp.mu.RLock()
	defer comp.mu.RUnlock()

	varKey := getVariableKey(variableName, variableInstance)
	variable := comp.Variables[varKey]
	if variable == nil {
		return "", GetVariableStatusUnknownVariable
	}

	variable.mu.RLock()
	defer variable.mu.RUnlock()

	// Default to Actual if no attribute type specified
	if attrType == "" {
		attrType = AttributeActual
	}

	attr := variable.Attributes[attrType]
	if attr == nil {
		return "", GetVariableStatusNotSupportedAttributeType
	}

	return attr.Value, GetVariableStatusAccepted
}

// SetVariable sets a variable value in a component
func (dm *DeviceModel) SetVariable(componentName, componentInstance, variableName, variableInstance string, attrType AttributeType, value string) SetVariableStatusType {
	comp := dm.GetComponent(componentName, componentInstance)
	if comp == nil {
		return SetVariableStatusUnknownComponent
	}

	comp.mu.Lock()
	defer comp.mu.Unlock()

	varKey := getVariableKey(variableName, variableInstance)
	variable := comp.Variables[varKey]
	if variable == nil {
		return SetVariableStatusUnknownVariable
	}

	variable.mu.Lock()
	defer variable.mu.Unlock()

	// Default to Actual if no attribute type specified
	if attrType == "" {
		attrType = AttributeActual
	}

	attr := variable.Attributes[attrType]
	if attr == nil {
		return SetVariableStatusNotSupportedAttributeType
	}

	// Check mutability
	if attr.Mutability == MutabilityReadOnly || attr.Constant {
		return SetVariableStatusRejected
	}

	// TODO: Add value validation based on characteristics
	attr.Value = value
	return SetVariableStatusAccepted
}

// AddVariable adds a variable to a component
func (comp *ComponentInstance) AddVariable(name, instance string, characteristics VariableCharacteristics) *VariableInstance {
	comp.mu.Lock()
	defer comp.mu.Unlock()

	key := getVariableKey(name, instance)
	v := &VariableInstance{
		Name:            name,
		Instance:        instance,
		Characteristics: characteristics,
		Attributes:      make(map[AttributeType]*VariableAttribute),
	}
	comp.Variables[key] = v
	return v
}

// SetAttribute sets an attribute for a variable
func (v *VariableInstance) SetAttribute(attrType AttributeType, value string, mutability Mutability, persistent, constant bool) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.Attributes[attrType] = &VariableAttribute{
		Type:       attrType,
		Value:      value,
		Mutability: mutability,
		Persistent: persistent,
		Constant:   constant,
	}
}

// GetAllVariables returns all variables for GetVariables response
func (dm *DeviceModel) GetAllVariables() []GetVariableResult {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var results []GetVariableResult

	for _, comp := range dm.Components {
		comp.mu.RLock()
		for _, variable := range comp.Variables {
			variable.mu.RLock()
			for attrType, attr := range variable.Attributes {
				attrTypeCopy := attrType // Create a copy for pointer
				result := GetVariableResult{
					AttributeStatus: GetVariableStatusAccepted,
					AttributeType:   &attrTypeCopy,
					AttributeValue:  attr.Value,
					Component: Component{
						Name:     comp.Name,
						Instance: comp.Instance,
						EVSE:     comp.EVSE,
					},
					Variable: Variable{
						Name:     variable.Name,
						Instance: variable.Instance,
					},
				}
				results = append(results, result)
			}
			variable.mu.RUnlock()
		}
		comp.mu.RUnlock()
	}

	return results
}

// initializeStandardComponents sets up the standard OCPP 2.0.1 device model components
func (dm *DeviceModel) initializeStandardComponents() {
	// ChargingStation component - main station configuration
	cs := dm.AddComponent("ChargingStation", "", nil)
	dm.addChargingStationVariables(cs)

	// SecurityCtrlr component - security settings
	sec := dm.AddComponent("SecurityCtrlr", "", nil)
	dm.addSecurityVariables(sec)

	// OCPPCommCtrlr component - OCPP communication settings
	ocpp := dm.AddComponent("OCPPCommCtrlr", "", nil)
	dm.addOCPPCommVariables(ocpp)

	// AuthCtrlr component - authorization settings
	auth := dm.AddComponent("AuthCtrlr", "", nil)
	dm.addAuthVariables(auth)

	// TxCtrlr component - transaction settings
	tx := dm.AddComponent("TxCtrlr", "", nil)
	dm.addTxVariables(tx)

	// SampledDataCtrlr component - metering settings
	sampled := dm.AddComponent("SampledDataCtrlr", "", nil)
	dm.addSampledDataVariables(sampled)

	// ClockCtrlr component - time settings
	clock := dm.AddComponent("ClockCtrlr", "", nil)
	dm.addClockVariables(clock)

	// DeviceDataCtrlr component - device information
	device := dm.AddComponent("DeviceDataCtrlr", "", nil)
	dm.addDeviceDataVariables(device)
}

// addChargingStationVariables adds variables for the ChargingStation component
func (dm *DeviceModel) addChargingStationVariables(comp *ComponentInstance) {
	// Model - charging station model
	model := comp.AddVariable("Model", "", VariableCharacteristics{
		DataType:        DataTypeString,
		SupportsMonitor: false,
	})
	model.SetAttribute(AttributeActual, "GenericCharger", MutabilityReadOnly, true, true)

	// VendorName - manufacturer name
	vendor := comp.AddVariable("VendorName", "", VariableCharacteristics{
		DataType:        DataTypeString,
		SupportsMonitor: false,
	})
	vendor.SetAttribute(AttributeActual, "GenericVendor", MutabilityReadOnly, true, true)

	// SerialNumber - serial number
	serial := comp.AddVariable("SerialNumber", "", VariableCharacteristics{
		DataType:        DataTypeString,
		SupportsMonitor: false,
	})
	serial.SetAttribute(AttributeActual, "SN-001", MutabilityReadOnly, true, true)

	// FirmwareVersion - firmware version
	fw := comp.AddVariable("FirmwareVersion", "", VariableCharacteristics{
		DataType:        DataTypeString,
		SupportsMonitor: false,
	})
	fw.SetAttribute(AttributeActual, "1.0.0", MutabilityReadOnly, true, false)

	// Available - station availability
	avail := comp.AddVariable("Available", "", VariableCharacteristics{
		DataType:        DataTypeBoolean,
		SupportsMonitor: true,
	})
	avail.SetAttribute(AttributeActual, "true", MutabilityReadWrite, true, false)

	// AvailabilityState - detailed availability state
	availState := comp.AddVariable("AvailabilityState", "", VariableCharacteristics{
		DataType:        DataTypeOptionList,
		SupportsMonitor: true,
		ValuesList:      "Available,Occupied,Reserved,Unavailable,Faulted",
	})
	availState.SetAttribute(AttributeActual, "Available", MutabilityReadOnly, false, false)
}

// addSecurityVariables adds variables for the SecurityCtrlr component
func (dm *DeviceModel) addSecurityVariables(comp *ComponentInstance) {
	// BasicAuthPassword - HTTP Basic Auth password
	basicAuth := comp.AddVariable("BasicAuthPassword", "", VariableCharacteristics{
		DataType:        DataTypeString,
		SupportsMonitor: false,
	})
	basicAuth.SetAttribute(AttributeActual, "", MutabilityWriteOnly, true, false)

	// SecurityProfile - active security profile
	secProfile := comp.AddVariable("SecurityProfile", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: true,
	})
	secProfile.SetAttribute(AttributeActual, "1", MutabilityReadWrite, true, false)

	// OrganizationName - organization name for certificates
	orgName := comp.AddVariable("OrganizationName", "", VariableCharacteristics{
		DataType:        DataTypeString,
		SupportsMonitor: false,
	})
	orgName.SetAttribute(AttributeActual, "OCPP Emulator", MutabilityReadWrite, true, false)
}

// addOCPPCommVariables adds variables for the OCPPCommCtrlr component
func (dm *DeviceModel) addOCPPCommVariables(comp *ComponentInstance) {
	// HeartbeatInterval - interval between heartbeats
	hbInterval := comp.AddVariable("HeartbeatInterval", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: true,
		Unit:            "s",
	})
	hbInterval.SetAttribute(AttributeActual, "60", MutabilityReadWrite, true, false)

	// RetryBackOffRepeatTimes - number of retry attempts
	retryTimes := comp.AddVariable("RetryBackOffRepeatTimes", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: false,
	})
	retryTimes.SetAttribute(AttributeActual, "3", MutabilityReadWrite, true, false)

	// RetryBackOffWaitMinimum - minimum wait time between retries
	retryMin := comp.AddVariable("RetryBackOffWaitMinimum", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: false,
		Unit:            "s",
	})
	retryMin.SetAttribute(AttributeActual, "10", MutabilityReadWrite, true, false)

	// NetworkConnectionProfiles - connection profiles
	netProfiles := comp.AddVariable("NetworkConnectionProfiles", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: false,
	})
	netProfiles.SetAttribute(AttributeActual, "1", MutabilityReadOnly, false, false)

	// WebSocketPingInterval - WebSocket ping interval
	wsPing := comp.AddVariable("WebSocketPingInterval", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: false,
		Unit:            "s",
	})
	wsPing.SetAttribute(AttributeActual, "30", MutabilityReadWrite, true, false)
}

// addAuthVariables adds variables for the AuthCtrlr component
func (dm *DeviceModel) addAuthVariables(comp *ComponentInstance) {
	// Enabled - authorization enabled
	enabled := comp.AddVariable("Enabled", "", VariableCharacteristics{
		DataType:        DataTypeBoolean,
		SupportsMonitor: true,
	})
	enabled.SetAttribute(AttributeActual, "true", MutabilityReadWrite, true, false)

	// LocalAuthorizeOffline - allow offline authorization
	localOffline := comp.AddVariable("LocalAuthorizeOffline", "", VariableCharacteristics{
		DataType:        DataTypeBoolean,
		SupportsMonitor: false,
	})
	localOffline.SetAttribute(AttributeActual, "true", MutabilityReadWrite, true, false)

	// LocalPreAuthorize - enable pre-authorization
	localPreAuth := comp.AddVariable("LocalPreAuthorize", "", VariableCharacteristics{
		DataType:        DataTypeBoolean,
		SupportsMonitor: false,
	})
	localPreAuth.SetAttribute(AttributeActual, "false", MutabilityReadWrite, true, false)

	// AuthorizeRemoteStart - require auth for remote start
	authRemote := comp.AddVariable("AuthorizeRemoteStart", "", VariableCharacteristics{
		DataType:        DataTypeBoolean,
		SupportsMonitor: false,
	})
	authRemote.SetAttribute(AttributeActual, "false", MutabilityReadWrite, true, false)
}

// addTxVariables adds variables for the TxCtrlr component
func (dm *DeviceModel) addTxVariables(comp *ComponentInstance) {
	// EVConnectionTimeOut - timeout for EV connection
	evTimeout := comp.AddVariable("EVConnectionTimeOut", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: false,
		Unit:            "s",
	})
	evTimeout.SetAttribute(AttributeActual, "30", MutabilityReadWrite, true, false)

	// StopTxOnEVSideDisconnect - stop transaction on disconnect
	stopOnDisconnect := comp.AddVariable("StopTxOnEVSideDisconnect", "", VariableCharacteristics{
		DataType:        DataTypeBoolean,
		SupportsMonitor: false,
	})
	stopOnDisconnect.SetAttribute(AttributeActual, "true", MutabilityReadWrite, true, false)

	// StopTxOnInvalidId - stop transaction on invalid ID
	stopOnInvalid := comp.AddVariable("StopTxOnInvalidId", "", VariableCharacteristics{
		DataType:        DataTypeBoolean,
		SupportsMonitor: false,
	})
	stopOnInvalid.SetAttribute(AttributeActual, "true", MutabilityReadWrite, true, false)

	// TxStartPoint - when to start transaction
	txStartPoint := comp.AddVariable("TxStartPoint", "", VariableCharacteristics{
		DataType:        DataTypeOptionList,
		SupportsMonitor: false,
		ValuesList:      "Authorized,EVConnected,PowerPathClosed",
	})
	txStartPoint.SetAttribute(AttributeActual, "Authorized", MutabilityReadWrite, true, false)

	// TxStopPoint - when to stop transaction
	txStopPoint := comp.AddVariable("TxStopPoint", "", VariableCharacteristics{
		DataType:        DataTypeOptionList,
		SupportsMonitor: false,
		ValuesList:      "EVConnected,Authorized,PowerPathClosed,EnergyTransfer",
	})
	txStopPoint.SetAttribute(AttributeActual, "EVConnected", MutabilityReadWrite, true, false)
}

// addSampledDataVariables adds variables for the SampledDataCtrlr component
func (dm *DeviceModel) addSampledDataVariables(comp *ComponentInstance) {
	// Enabled - sampling enabled
	enabled := comp.AddVariable("Enabled", "", VariableCharacteristics{
		DataType:        DataTypeBoolean,
		SupportsMonitor: false,
	})
	enabled.SetAttribute(AttributeActual, "true", MutabilityReadWrite, true, false)

	// TxUpdatedInterval - interval for transaction updates
	txInterval := comp.AddVariable("TxUpdatedInterval", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: false,
		Unit:            "s",
	})
	txInterval.SetAttribute(AttributeActual, "60", MutabilityReadWrite, true, false)

	// TxUpdatedMeasurands - measurands to include in updates
	txMeasurands := comp.AddVariable("TxUpdatedMeasurands", "", VariableCharacteristics{
		DataType:        DataTypeMemberList,
		SupportsMonitor: false,
		ValuesList:      "Energy.Active.Import.Register,Power.Active.Import,Current.Import,Voltage",
	})
	txMeasurands.SetAttribute(AttributeActual, "Energy.Active.Import.Register,Power.Active.Import", MutabilityReadWrite, true, false)

	// TxEndedMeasurands - measurands for transaction end
	txEndMeasurands := comp.AddVariable("TxEndedMeasurands", "", VariableCharacteristics{
		DataType:        DataTypeMemberList,
		SupportsMonitor: false,
		ValuesList:      "Energy.Active.Import.Register,Power.Active.Import,SoC",
	})
	txEndMeasurands.SetAttribute(AttributeActual, "Energy.Active.Import.Register", MutabilityReadWrite, true, false)
}

// addClockVariables adds variables for the ClockCtrlr component
func (dm *DeviceModel) addClockVariables(comp *ComponentInstance) {
	// TimeSource - time synchronization source
	timeSource := comp.AddVariable("TimeSource", "", VariableCharacteristics{
		DataType:        DataTypeOptionList,
		SupportsMonitor: false,
		ValuesList:      "Heartbeat,SNTP,GPS,RTC",
	})
	timeSource.SetAttribute(AttributeActual, "Heartbeat", MutabilityReadWrite, true, false)

	// TimeOffset - timezone offset
	timeOffset := comp.AddVariable("TimeOffset", "", VariableCharacteristics{
		DataType:        DataTypeString,
		SupportsMonitor: false,
	})
	timeOffset.SetAttribute(AttributeActual, "+00:00", MutabilityReadWrite, true, false)

	// NtpServerUri - NTP server address
	ntpServer := comp.AddVariable("NtpServerUri", "", VariableCharacteristics{
		DataType:        DataTypeString,
		SupportsMonitor: false,
	})
	ntpServer.SetAttribute(AttributeActual, "", MutabilityReadWrite, true, false)
}

// addDeviceDataVariables adds variables for the DeviceDataCtrlr component
func (dm *DeviceModel) addDeviceDataVariables(comp *ComponentInstance) {
	// BytesPerMessage - max bytes per message
	bytesPerMsg := comp.AddVariable("BytesPerMessage", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: false,
	})
	bytesPerMsg.SetAttribute(AttributeActual, "65535", MutabilityReadOnly, false, true)

	// ItemsPerMessage - max items per message
	itemsPerMsg := comp.AddVariable("ItemsPerMessage", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: false,
	})
	itemsPerMsg.SetAttribute(AttributeActual, "100", MutabilityReadOnly, false, true)

	// ConfigurationValueSize - max config value size
	configSize := comp.AddVariable("ConfigurationValueSize", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: false,
	})
	configSize.SetAttribute(AttributeActual, "1000", MutabilityReadOnly, false, true)
}

// AddEVSEComponent adds an EVSE component with standard variables
func (dm *DeviceModel) AddEVSEComponent(evseID int) *ComponentInstance {
	evse := &EVSE{ID: evseID}
	comp := dm.AddComponent("EVSE", fmt.Sprintf("%d", evseID), evse)

	// Available - EVSE availability
	avail := comp.AddVariable("Available", "", VariableCharacteristics{
		DataType:        DataTypeBoolean,
		SupportsMonitor: true,
	})
	avail.SetAttribute(AttributeActual, "true", MutabilityReadWrite, true, false)

	// AvailabilityState - EVSE state
	availState := comp.AddVariable("AvailabilityState", "", VariableCharacteristics{
		DataType:        DataTypeOptionList,
		SupportsMonitor: true,
		ValuesList:      "Available,Occupied,Reserved,Unavailable,Faulted",
	})
	availState.SetAttribute(AttributeActual, "Available", MutabilityReadOnly, false, false)

	// Power - max power
	power := comp.AddVariable("Power", "", VariableCharacteristics{
		DataType:        DataTypeDecimal,
		SupportsMonitor: true,
		Unit:            "W",
	})
	power.SetAttribute(AttributeActual, "22000", MutabilityReadOnly, false, false)
	power.SetAttribute(AttributeMaxSet, "22000", MutabilityReadOnly, true, true)

	// SupplyPhases - number of phases
	phases := comp.AddVariable("SupplyPhases", "", VariableCharacteristics{
		DataType:        DataTypeInteger,
		SupportsMonitor: false,
	})
	phases.SetAttribute(AttributeActual, "3", MutabilityReadOnly, true, false)

	return comp
}

// AddConnectorComponent adds a Connector component with standard variables
func (dm *DeviceModel) AddConnectorComponent(evseID, connectorID int, connectorType string) *ComponentInstance {
	evse := &EVSE{ID: evseID, ConnectorId: &connectorID}
	comp := dm.AddComponent("Connector", fmt.Sprintf("%d:%d", evseID, connectorID), evse)

	// Available - connector availability
	avail := comp.AddVariable("Available", "", VariableCharacteristics{
		DataType:        DataTypeBoolean,
		SupportsMonitor: true,
	})
	avail.SetAttribute(AttributeActual, "true", MutabilityReadWrite, true, false)

	// AvailabilityState - connector state
	availState := comp.AddVariable("AvailabilityState", "", VariableCharacteristics{
		DataType:        DataTypeOptionList,
		SupportsMonitor: true,
		ValuesList:      "Available,Occupied,Reserved,Unavailable,Faulted",
	})
	availState.SetAttribute(AttributeActual, "Available", MutabilityReadOnly, false, false)

	// ConnectorType - type of connector
	connType := comp.AddVariable("ConnectorType", "", VariableCharacteristics{
		DataType:        DataTypeOptionList,
		SupportsMonitor: false,
		ValuesList:      "cCCS1,cCCS2,cG105,cTesla,cType1,cType2,s309-1P-16A,s309-1P-32A,s309-3P-16A,s309-3P-32A,sBS1361,sCEE-7-7,sType2,sType3,Other1PhMax16A,Other1PhOver16A,Other3Ph,Pan,wInductive,wResonant,Undetermined,Unknown",
	})
	connType.SetAttribute(AttributeActual, connectorType, MutabilityReadOnly, true, true)

	return comp
}

// UpdateStationInfo updates the ChargingStation component with actual station info
func (dm *DeviceModel) UpdateStationInfo(vendor, model, serialNumber, firmwareVersion string) {
	comp := dm.GetComponent("ChargingStation", "")
	if comp == nil {
		return
	}

	comp.mu.Lock()
	defer comp.mu.Unlock()

	if v := comp.Variables["Model"]; v != nil {
		v.mu.Lock()
		if attr := v.Attributes[AttributeActual]; attr != nil {
			attr.Value = model
		}
		v.mu.Unlock()
	}

	if v := comp.Variables["VendorName"]; v != nil {
		v.mu.Lock()
		if attr := v.Attributes[AttributeActual]; attr != nil {
			attr.Value = vendor
		}
		v.mu.Unlock()
	}

	if v := comp.Variables["SerialNumber"]; v != nil {
		v.mu.Lock()
		if attr := v.Attributes[AttributeActual]; attr != nil {
			attr.Value = serialNumber
		}
		v.mu.Unlock()
	}

	if v := comp.Variables["FirmwareVersion"]; v != nil {
		v.mu.Lock()
		if attr := v.Attributes[AttributeActual]; attr != nil {
			attr.Value = firmwareVersion
		}
		v.mu.Unlock()
	}
}
