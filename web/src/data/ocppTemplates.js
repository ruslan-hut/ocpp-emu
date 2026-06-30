// OCPP message templates for the Message Crafter, keyed by action name.
// Built fresh per call so timestamp fields reflect the current time when a
// template is selected.

function ocpp16Templates() {
  return {
    Heartbeat: '{}',
    BootNotification: JSON.stringify({
      chargePointVendor: "VendorName",
      chargePointModel: "ModelX"
    }, null, 2),
    StatusNotification: JSON.stringify({
      connectorId: 1,
      errorCode: "NoError",
      status: "Available"
    }, null, 2),
    Authorize: JSON.stringify({
      idTag: "TAG123456"
    }, null, 2),
    StartTransaction: JSON.stringify({
      connectorId: 1,
      idTag: "TAG123456",
      meterStart: 0,
      timestamp: new Date().toISOString()
    }, null, 2),
    StopTransaction: JSON.stringify({
      transactionId: 1,
      meterStop: 1000,
      timestamp: new Date().toISOString()
    }, null, 2),
    MeterValues: JSON.stringify({
      connectorId: 1,
      transactionId: 1,
      meterValue: [{
        timestamp: new Date().toISOString(),
        sampledValue: [{
          value: "1000",
          context: "Sample.Periodic",
          measurand: "Energy.Active.Import.Register",
          unit: "Wh"
        }]
      }]
    }, null, 2),
    DataTransfer: JSON.stringify({
      vendorId: "VendorName",
      messageId: "CustomMessage",
      data: "test data"
    }, null, 2)
  }
}

function ocpp201Templates() {
  return {
    Heartbeat: '{}',
    BootNotification: JSON.stringify({
      reason: "PowerUp",
      chargingStation: {
        model: "ModelX",
        vendorName: "VendorName",
        serialNumber: "SN123456",
        firmwareVersion: "1.0.0"
      }
    }, null, 2),
    StatusNotification: JSON.stringify({
      timestamp: new Date().toISOString(),
      connectorStatus: "Available",
      evseId: 1,
      connectorId: 1
    }, null, 2),
    Authorize: JSON.stringify({
      idToken: {
        idToken: "TAG123456",
        type: "ISO14443"
      }
    }, null, 2),
    TransactionEvent: JSON.stringify({
      eventType: "Started",
      timestamp: new Date().toISOString(),
      triggerReason: "Authorized",
      seqNo: 0,
      transactionInfo: {
        transactionId: "TX-" + Date.now(),
        chargingState: "Charging"
      },
      idToken: {
        idToken: "TAG123456",
        type: "ISO14443"
      },
      evse: {
        id: 1,
        connectorId: 1
      }
    }, null, 2),
    NotifyReport: JSON.stringify({
      requestId: 1,
      generatedAt: new Date().toISOString(),
      seqNo: 0,
      reportData: [{
        component: {
          name: "ChargingStation"
        },
        variable: {
          name: "Model"
        },
        variableAttribute: [{
          type: "Actual",
          value: "ModelX",
          mutability: "ReadOnly"
        }]
      }]
    }, null, 2),
    GetVariables: JSON.stringify({
      getVariableData: [{
        component: {
          name: "ChargingStation"
        },
        variable: {
          name: "Model"
        },
        attributeType: "Actual"
      }]
    }, null, 2),
    SetVariables: JSON.stringify({
      setVariableData: [{
        component: {
          name: "ChargingStation"
        },
        variable: {
          name: "AllowNewSessionsPendingFirmwareUpdate"
        },
        attributeType: "Actual",
        attributeValue: "true"
      }]
    }, null, 2),
    SignCertificate: JSON.stringify({
      csr: "-----BEGIN CERTIFICATE REQUEST-----\nMIIBIjANBgkqh...\n-----END CERTIFICATE REQUEST-----",
      certificateType: "ChargingStationCertificate"
    }, null, 2),
    Get15118EVCertificate: JSON.stringify({
      iso15118SchemaVersion: "urn:iso:15118:2:2013:MsgDef",
      action: "Install",
      exiRequest: "base64-encoded-exi-request"
    }, null, 2),
    SecurityEventNotification: JSON.stringify({
      type: "FirmwareUpdated",
      timestamp: new Date().toISOString(),
      techInfo: "Firmware updated to version 1.1.0"
    }, null, 2),
    DataTransfer: JSON.stringify({
      vendorId: "VendorName",
      messageId: "CustomMessage",
      data: "test data"
    }, null, 2),
    LogStatusNotification: JSON.stringify({
      status: "Uploaded",
      requestId: 1
    }, null, 2),
    FirmwareStatusNotification: JSON.stringify({
      status: "Installed",
      requestId: 1
    }, null, 2)
  }
}

function ocpp21Templates() {
  return {
    // Inherited from 2.0.1
    ...ocpp201Templates(),

    // OCPP 2.1 Cost and Tariff
    CostUpdated: JSON.stringify({
      totalCost: 12.50,
      transactionId: "TX-123456"
    }, null, 2),
    NotifyCustomerInformation: JSON.stringify({
      data: "Customer information data",
      seqNo: 0,
      requestId: 1,
      tbc: false,
      generatedAt: new Date().toISOString()
    }, null, 2),
    NotifyEVChargingNeeds: JSON.stringify({
      evseId: 1,
      chargingNeeds: {
        requestedEnergyTransfer: "DC",
        departureTime: new Date(Date.now() + 3600000).toISOString(),
        dcChargingParameters: {
          evMaxCurrent: 300,
          evMaxVoltage: 500,
          evMaxPower: 150000,
          stateOfCharge: 20,
          evEnergyCapacity: 75000
        }
      }
    }, null, 2),

    // OCPP 2.1 Display Messages
    SetDisplayMessage: JSON.stringify({
      message: {
        id: 1,
        priority: "NormalCycle",
        state: "Idle",
        message: {
          format: "UTF8",
          content: "Welcome to the charging station"
        }
      }
    }, null, 2),
    GetDisplayMessages: JSON.stringify({
      requestId: 1,
      priority: "NormalCycle"
    }, null, 2),
    ClearDisplayMessage: JSON.stringify({
      id: 1
    }, null, 2),
    NotifyDisplayMessages: JSON.stringify({
      requestId: 1,
      tbc: false,
      messageInfo: [{
        id: 1,
        priority: "NormalCycle",
        message: {
          format: "UTF8",
          content: "Station message"
        }
      }]
    }, null, 2),

    // OCPP 2.1 Reservations
    ReserveNow: JSON.stringify({
      id: 1,
      expiryDateTime: new Date(Date.now() + 3600000).toISOString(),
      evseId: 1,
      idToken: {
        idToken: "TAG123456",
        type: "ISO14443"
      }
    }, null, 2),
    CancelReservation: JSON.stringify({
      reservationId: 1
    }, null, 2),

    // OCPP 2.1 Charging Profiles
    SetChargingProfile: JSON.stringify({
      evseId: 1,
      chargingProfile: {
        id: 1,
        stackLevel: 0,
        chargingProfilePurpose: "TxDefaultProfile",
        chargingProfileKind: "Relative",
        chargingSchedule: [{
          id: 1,
          chargingRateUnit: "W",
          chargingSchedulePeriod: [{
            startPeriod: 0,
            limit: 11000
          }]
        }]
      }
    }, null, 2),
    GetChargingProfiles: JSON.stringify({
      requestId: 1,
      evseId: 1,
      chargingProfile: {
        chargingProfilePurpose: "TxDefaultProfile"
      }
    }, null, 2),
    ClearChargingProfile: JSON.stringify({
      chargingProfileId: 1
    }, null, 2),
    GetCompositeSchedule: JSON.stringify({
      evseId: 1,
      duration: 3600,
      chargingRateUnit: "W"
    }, null, 2),
    ReportChargingProfiles: JSON.stringify({
      requestId: 1,
      chargingLimitSource: "EMS",
      evseId: 1,
      tbc: false,
      chargingProfile: [{
        id: 1,
        stackLevel: 0,
        chargingProfilePurpose: "TxDefaultProfile",
        chargingProfileKind: "Relative",
        chargingSchedule: [{
          id: 1,
          chargingRateUnit: "W",
          chargingSchedulePeriod: [{
            startPeriod: 0,
            limit: 11000
          }]
        }]
      }]
    }, null, 2),
    NotifyChargingLimit: JSON.stringify({
      chargingLimit: {
        chargingLimitSource: "EMS",
        isGridCritical: false
      },
      chargingSchedule: [{
        id: 1,
        chargingRateUnit: "W",
        chargingSchedulePeriod: [{
          startPeriod: 0,
          limit: 11000
        }]
      }]
    }, null, 2),
    ClearedChargingLimit: JSON.stringify({
      chargingLimitSource: "EMS",
      evseId: 1
    }, null, 2),

    // OCPP 2.1 Local Authorization
    GetLocalListVersion: JSON.stringify({}, null, 2),
    SendLocalList: JSON.stringify({
      versionNumber: 1,
      updateType: "Full",
      localAuthorizationList: [{
        idToken: {
          idToken: "TAG123456",
          type: "ISO14443"
        },
        idTokenInfo: {
          status: "Accepted"
        }
      }]
    }, null, 2),

    // OCPP 2.1 Firmware Management
    UpdateFirmware: JSON.stringify({
      requestId: 1,
      firmware: {
        location: "https://example.com/firmware/v1.2.0.bin",
        retrieveDateTime: new Date().toISOString(),
        signingCertificate: "",
        signature: ""
      },
      retries: 3,
      retryInterval: 60
    }, null, 2),
    SetNetworkProfile: JSON.stringify({
      configurationSlot: 1,
      connectionData: {
        ocppVersion: "OCPP21",
        ocppTransport: "JSON",
        ocppCsmsUrl: "wss://csms.example.com/ocpp",
        messageTimeout: 30,
        securityProfile: 1,
        ocppInterface: "Wired0"
      }
    }, null, 2),
    GetLog: JSON.stringify({
      logType: "DiagnosticsLog",
      requestId: 1,
      log: {
        remoteLocation: "https://example.com/logs/upload"
      }
    }, null, 2)
  }
}

// getMessageTemplates returns the action→payload template map for the given
// protocol flags, matching the precedence used by the Message Crafter.
export function getMessageTemplates({ isOcpp21, isOcpp201 }) {
  if (isOcpp21) return ocpp21Templates()
  if (isOcpp201) return ocpp201Templates()
  return ocpp16Templates()
}
