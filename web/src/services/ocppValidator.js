// OCPP Message Validation Service (1.6 and 2.0.1)

// OCPP Protocol versions
export const OCPPProtocol = {
  OCPP16: 'ocpp1.6',
  OCPP201: 'ocpp2.0.1',
  OCPP21: 'ocpp2.1'
}

// OCPP 1.6 Action Schemas
const OCPP16_SCHEMAS = {
  // Core Profile - Charge Point Initiated
  Heartbeat: {
    required: [],
    optional: []
  },

  BootNotification: {
    required: ['chargePointVendor', 'chargePointModel'],
    optional: ['chargePointSerialNumber', 'chargeBoxSerialNumber', 'firmwareVersion', 'iccid', 'imsi', 'meterType', 'meterSerialNumber'],
    types: {
      chargePointVendor: 'string',
      chargePointModel: 'string',
      chargePointSerialNumber: 'string',
      chargeBoxSerialNumber: 'string',
      firmwareVersion: 'string',
      iccid: 'string',
      imsi: 'string',
      meterType: 'string',
      meterSerialNumber: 'string'
    }
  },

  StatusNotification: {
    required: ['connectorId', 'errorCode', 'status'],
    optional: ['timestamp', 'info', 'vendorId', 'vendorErrorCode'],
    types: {
      connectorId: 'number',
      errorCode: 'string',
      status: 'string',
      timestamp: 'string',
      info: 'string',
      vendorId: 'string',
      vendorErrorCode: 'string'
    },
    enums: {
      errorCode: ['ConnectorLockFailure', 'EVCommunicationError', 'GroundFailure', 'HighTemperature', 'InternalError', 'LocalListConflict', 'NoError', 'OtherError', 'OverCurrentFailure', 'PowerMeterFailure', 'PowerSwitchFailure', 'ReaderFailure', 'ResetFailure', 'UnderVoltage', 'OverVoltage', 'WeakSignal'],
      status: ['Available', 'Preparing', 'Charging', 'SuspendedEVSE', 'SuspendedEV', 'Finishing', 'Reserved', 'Unavailable', 'Faulted']
    }
  },

  Authorize: {
    required: ['idTag'],
    optional: [],
    types: {
      idTag: 'string'
    }
  },

  StartTransaction: {
    required: ['connectorId', 'idTag', 'meterStart', 'timestamp'],
    optional: ['reservationId'],
    types: {
      connectorId: 'number',
      idTag: 'string',
      meterStart: 'number',
      timestamp: 'string',
      reservationId: 'number'
    }
  },

  StopTransaction: {
    required: ['meterStop', 'timestamp', 'transactionId'],
    optional: ['idTag', 'reason', 'transactionData'],
    types: {
      meterStop: 'number',
      timestamp: 'string',
      transactionId: 'number',
      idTag: 'string',
      reason: 'string'
    },
    enums: {
      reason: ['EmergencyStop', 'EVDisconnected', 'HardReset', 'Local', 'Other', 'PowerLoss', 'Reboot', 'Remote', 'SoftReset', 'UnlockCommand', 'DeAuthorized']
    }
  },

  MeterValues: {
    required: ['connectorId', 'meterValue'],
    optional: ['transactionId'],
    types: {
      connectorId: 'number',
      transactionId: 'number',
      meterValue: 'array'
    }
  },

  DataTransfer: {
    required: ['vendorId'],
    optional: ['messageId', 'data'],
    types: {
      vendorId: 'string',
      messageId: 'string',
      data: 'string'
    }
  },

  DiagnosticsStatusNotification: {
    required: ['status'],
    optional: [],
    types: {
      status: 'string'
    },
    enums: {
      status: ['Idle', 'Uploaded', 'UploadFailed', 'Uploading']
    }
  },

  FirmwareStatusNotification: {
    required: ['status'],
    optional: [],
    types: {
      status: 'string'
    },
    enums: {
      status: ['Downloaded', 'DownloadFailed', 'Downloading', 'Idle', 'InstallationFailed', 'Installing', 'Installed']
    }
  }
}

// OCPP 2.0.1 Action Schemas
const OCPP201_SCHEMAS = {
  Heartbeat: {
    required: [],
    optional: []
  },

  BootNotification: {
    required: ['reason', 'chargingStation'],
    optional: [],
    types: {
      reason: 'string',
      chargingStation: 'object'
    },
    enums: {
      reason: ['ApplicationReset', 'FirmwareUpdate', 'LocalReset', 'PowerUp', 'RemoteReset', 'ScheduledReset', 'Triggered', 'Unknown', 'Watchdog']
    },
    nested: {
      chargingStation: {
        required: ['model', 'vendorName'],
        optional: ['serialNumber', 'modem', 'firmwareVersion'],
        types: {
          model: 'string',
          vendorName: 'string',
          serialNumber: 'string',
          firmwareVersion: 'string',
          modem: 'object'
        }
      }
    }
  },

  StatusNotification: {
    required: ['timestamp', 'connectorStatus', 'evseId', 'connectorId'],
    optional: [],
    types: {
      timestamp: 'string',
      connectorStatus: 'string',
      evseId: 'number',
      connectorId: 'number'
    },
    enums: {
      connectorStatus: ['Available', 'Occupied', 'Reserved', 'Unavailable', 'Faulted']
    }
  },

  Authorize: {
    required: ['idToken'],
    optional: ['certificate', 'iso15118CertificateHashData'],
    types: {
      idToken: 'object',
      certificate: 'string',
      iso15118CertificateHashData: 'array'
    },
    nested: {
      idToken: {
        required: ['idToken', 'type'],
        optional: ['additionalInfo'],
        types: {
          idToken: 'string',
          type: 'string',
          additionalInfo: 'array'
        },
        enums: {
          type: ['Central', 'eMAID', 'ISO14443', 'ISO15693', 'KeyCode', 'Local', 'MacAddress', 'NoAuthorization']
        }
      }
    }
  },

  TransactionEvent: {
    required: ['eventType', 'timestamp', 'triggerReason', 'seqNo', 'transactionInfo'],
    optional: ['meterValue', 'idToken', 'evse', 'offline', 'numberOfPhasesUsed', 'cableMaxCurrent', 'reservationId'],
    types: {
      eventType: 'string',
      timestamp: 'string',
      triggerReason: 'string',
      seqNo: 'number',
      transactionInfo: 'object',
      meterValue: 'array',
      idToken: 'object',
      evse: 'object',
      offline: 'boolean',
      numberOfPhasesUsed: 'number',
      cableMaxCurrent: 'number',
      reservationId: 'number'
    },
    enums: {
      eventType: ['Ended', 'Started', 'Updated'],
      triggerReason: ['Authorized', 'CablePluggedIn', 'ChargingRateChanged', 'ChargingStateChanged', 'Deauthorized', 'EnergyLimitReached', 'EVCommunicationLost', 'EVConnectTimeout', 'MeterValueClock', 'MeterValuePeriodic', 'TimeLimitReached', 'Trigger', 'UnlockCommand', 'StopAuthorized', 'EVDeparted', 'EVDetected', 'RemoteStop', 'RemoteStart', 'AbnormalCondition', 'SignedDataReceived', 'ResetCommand']
    },
    nested: {
      transactionInfo: {
        required: ['transactionId'],
        optional: ['chargingState', 'timeSpentCharging', 'stoppedReason', 'remoteStartId'],
        types: {
          transactionId: 'string',
          chargingState: 'string',
          timeSpentCharging: 'number',
          stoppedReason: 'string',
          remoteStartId: 'number'
        },
        enums: {
          chargingState: ['Charging', 'EVConnected', 'SuspendedEV', 'SuspendedEVSE', 'Idle'],
          stoppedReason: ['DeAuthorized', 'EmergencyStop', 'EnergyLimitReached', 'EVDisconnected', 'GroundFault', 'ImmediateReset', 'Local', 'LocalOutOfCredit', 'MasterPass', 'Other', 'OvercurrentFault', 'PowerLoss', 'PowerQuality', 'Reboot', 'Remote', 'SOCLimitReached', 'StoppedByEV', 'TimeLimitReached', 'Timeout']
        }
      },
      evse: {
        required: ['id'],
        optional: ['connectorId'],
        types: {
          id: 'number',
          connectorId: 'number'
        }
      }
    }
  },

  NotifyReport: {
    required: ['requestId', 'generatedAt', 'seqNo'],
    optional: ['reportData', 'tbc'],
    types: {
      requestId: 'number',
      generatedAt: 'string',
      seqNo: 'number',
      reportData: 'array',
      tbc: 'boolean'
    }
  },

  GetVariables: {
    required: ['getVariableData'],
    optional: [],
    types: {
      getVariableData: 'array'
    }
  },

  SetVariables: {
    required: ['setVariableData'],
    optional: [],
    types: {
      setVariableData: 'array'
    }
  },

  SignCertificate: {
    required: ['csr'],
    optional: ['certificateType'],
    types: {
      csr: 'string',
      certificateType: 'string'
    },
    enums: {
      certificateType: ['ChargingStationCertificate', 'V2GCertificate']
    }
  },

  Get15118EVCertificate: {
    required: ['iso15118SchemaVersion', 'action', 'exiRequest'],
    optional: [],
    types: {
      iso15118SchemaVersion: 'string',
      action: 'string',
      exiRequest: 'string'
    },
    enums: {
      action: ['Install', 'Update']
    }
  },

  SecurityEventNotification: {
    required: ['type', 'timestamp'],
    optional: ['techInfo'],
    types: {
      type: 'string',
      timestamp: 'string',
      techInfo: 'string'
    }
  },

  DataTransfer: {
    required: ['vendorId'],
    optional: ['messageId', 'data'],
    types: {
      vendorId: 'string',
      messageId: 'string',
      data: 'string'
    }
  },

  LogStatusNotification: {
    required: ['status'],
    optional: ['requestId'],
    types: {
      status: 'string',
      requestId: 'number'
    },
    enums: {
      status: ['BadMessage', 'Idle', 'NotSupportedOperation', 'PermissionDenied', 'Uploaded', 'UploadFailure', 'Uploading', 'AcceptedCanceled']
    }
  },

  FirmwareStatusNotification: {
    required: ['status'],
    optional: ['requestId'],
    types: {
      status: 'string',
      requestId: 'number'
    },
    enums: {
      status: ['Downloaded', 'DownloadFailed', 'Downloading', 'DownloadScheduled', 'DownloadPaused', 'Idle', 'InstallationFailed', 'Installing', 'Installed', 'InstallRebooting', 'InstallScheduled', 'InstallVerificationFailed', 'InvalidSignature', 'SignatureVerified']
    }
  }
}

// Validation modes
export const ValidationMode = {
  STRICT: 'strict',   // Enforce full OCPP spec compliance
  LENIENT: 'lenient'  // Allow testing of invalid messages
}

export class OCPPValidator {
  constructor(mode = ValidationMode.STRICT, protocol = OCPPProtocol.OCPP16) {
    this.mode = mode
    this.protocol = protocol
  }

  setMode(mode) {
    this.mode = mode
  }

  setProtocol(protocol) {
    this.protocol = protocol
  }

  getSchemas() {
    if (this.protocol === OCPPProtocol.OCPP201 || this.protocol === OCPPProtocol.OCPP21) {
      return OCPP201_SCHEMAS
    }
    return OCPP16_SCHEMAS
  }

  /**
   * Validate a complete OCPP message
   * @param {Array} message - OCPP message array
   * @returns {Object} { valid: boolean, errors: [], warnings: [] }
   */
  validateMessage(message) {
    const errors = []
    const warnings = []

    // Check if message is an array
    if (!Array.isArray(message)) {
      errors.push({
        field: 'message',
        message: 'OCPP message must be an array',
        severity: 'error'
      })
      return { valid: false, errors, warnings }
    }

    // Check message type
    const messageType = message[0]
    if (![2, 3, 4].includes(messageType)) {
      errors.push({
        field: 'messageType',
        message: `Invalid message type: ${messageType}. Must be 2 (Call), 3 (CallResult), or 4 (CallError)`,
        severity: 'error'
      })
      return { valid: false, errors, warnings }
    }

    // Validate based on message type
    if (messageType === 2) {
      // Call message: [2, uniqueId, action, payload]
      this.validateCallMessage(message, errors, warnings)
    } else if (messageType === 3) {
      // CallResult message: [3, uniqueId, payload]
      this.validateCallResultMessage(message, errors, warnings)
    } else if (messageType === 4) {
      // CallError message: [4, uniqueId, errorCode, errorDescription, errorDetails]
      this.validateCallErrorMessage(message, errors, warnings)
    }

    return {
      valid: errors.length === 0 && (this.mode === ValidationMode.STRICT ? warnings.length === 0 : true),
      errors,
      warnings
    }
  }

  validateCallMessage(message, errors, warnings) {
    // Check structure
    if (message.length !== 4) {
      errors.push({
        field: 'message',
        message: `Call message must have 4 elements, got ${message.length}`,
        severity: 'error'
      })
      return
    }

    const [messageType, uniqueId, action, payload] = message

    // Validate uniqueId
    if (typeof uniqueId !== 'string' || uniqueId.trim() === '') {
      errors.push({
        field: 'uniqueId',
        message: 'uniqueId must be a non-empty string',
        severity: 'error'
      })
    }

    // Validate action
    if (typeof action !== 'string' || action.trim() === '') {
      errors.push({
        field: 'action',
        message: 'action must be a non-empty string',
        severity: 'error'
      })
      return
    }

    // Validate payload is an object
    if (typeof payload !== 'object' || payload === null || Array.isArray(payload)) {
      errors.push({
        field: 'payload',
        message: 'payload must be an object',
        severity: 'error'
      })
      return
    }

    // Validate payload against schema if available
    const schemas = this.getSchemas()
    const schema = schemas[action]
    if (!schema) {
      warnings.push({
        field: 'action',
        message: `No validation schema found for action: ${action} (${this.protocol}). Cannot validate payload.`,
        severity: 'warning'
      })
      return
    }

    this.validatePayload(action, payload, schema, errors, warnings)
  }

  validateCallResultMessage(message, errors, warnings) {
    if (message.length !== 3) {
      errors.push({
        field: 'message',
        message: `CallResult message must have 3 elements, got ${message.length}`,
        severity: 'error'
      })
      return
    }

    const [messageType, uniqueId, payload] = message

    if (typeof uniqueId !== 'string' || uniqueId.trim() === '') {
      errors.push({
        field: 'uniqueId',
        message: 'uniqueId must be a non-empty string',
        severity: 'error'
      })
    }

    if (typeof payload !== 'object' || payload === null || Array.isArray(payload)) {
      errors.push({
        field: 'payload',
        message: 'payload must be an object',
        severity: 'error'
      })
    }
  }

  validateCallErrorMessage(message, errors, warnings) {
    if (message.length !== 5) {
      errors.push({
        field: 'message',
        message: `CallError message must have 5 elements, got ${message.length}`,
        severity: 'error'
      })
      return
    }

    const [messageType, uniqueId, errorCode, errorDescription, errorDetails] = message

    if (typeof uniqueId !== 'string' || uniqueId.trim() === '') {
      errors.push({
        field: 'uniqueId',
        message: 'uniqueId must be a non-empty string',
        severity: 'error'
      })
    }

    if (typeof errorCode !== 'string' || errorCode.trim() === '') {
      errors.push({
        field: 'errorCode',
        message: 'errorCode must be a non-empty string',
        severity: 'error'
      })
    }

    const validErrorCodes = [
      'NotImplemented',
      'NotSupported',
      'InternalError',
      'ProtocolError',
      'SecurityError',
      'FormationViolation',
      'PropertyConstraintViolation',
      'OccurenceConstraintViolation',
      'TypeConstraintViolation',
      'GenericError'
    ]

    if (!validErrorCodes.includes(errorCode)) {
      warnings.push({
        field: 'errorCode',
        message: `errorCode '${errorCode}' is not a standard OCPP error code`,
        severity: 'warning'
      })
    }

    if (typeof errorDescription !== 'string') {
      errors.push({
        field: 'errorDescription',
        message: 'errorDescription must be a string',
        severity: 'error'
      })
    }

    if (typeof errorDetails !== 'object' || errorDetails === null || Array.isArray(errorDetails)) {
      errors.push({
        field: 'errorDetails',
        message: 'errorDetails must be an object',
        severity: 'error'
      })
    }
  }

  validatePayload(action, payload, schema, errors, warnings) {
    const payloadKeys = Object.keys(payload)

    // Check required fields
    schema.required.forEach(field => {
      if (!(field in payload)) {
        errors.push({
          field: `payload.${field}`,
          message: `Required field '${field}' is missing for action ${action}`,
          severity: 'error'
        })
      }
    })

    // Check for unknown fields
    const allowedFields = [...schema.required, ...schema.optional]
    payloadKeys.forEach(key => {
      if (!allowedFields.includes(key)) {
        if (this.mode === ValidationMode.STRICT) {
          errors.push({
            field: `payload.${key}`,
            message: `Unknown field '${key}' for action ${action}`,
            severity: 'error'
          })
        } else {
          warnings.push({
            field: `payload.${key}`,
            message: `Unknown field '${key}' for action ${action}`,
            severity: 'warning'
          })
        }
      }
    })

    // Validate field types
    if (schema.types) {
      Object.keys(payload).forEach(key => {
        if (schema.types[key]) {
          const expectedType = schema.types[key]
          const actualType = this.getType(payload[key])

          if (expectedType !== actualType) {
            errors.push({
              field: `payload.${key}`,
              message: `Field '${key}' should be ${expectedType}, got ${actualType}`,
              severity: 'error'
            })
          }
        }
      })
    }

    // Validate enumerations
    if (schema.enums) {
      Object.keys(payload).forEach(key => {
        if (schema.enums[key]) {
          const validValues = schema.enums[key]
          const actualValue = payload[key]

          if (!validValues.includes(actualValue)) {
            errors.push({
              field: `payload.${key}`,
              message: `Field '${key}' must be one of: ${validValues.join(', ')}. Got: ${actualValue}`,
              severity: 'error'
            })
          }
        }
      })
    }

    // Validate timestamp format
    Object.keys(payload).forEach(key => {
      if (key === 'timestamp' || key.endsWith('Timestamp')) {
        const timestamp = payload[key]
        if (typeof timestamp === 'string') {
          const iso8601Regex = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{3})?Z?$/
          if (!iso8601Regex.test(timestamp)) {
            warnings.push({
              field: `payload.${key}`,
              message: `Field '${key}' should be in ISO 8601 format (e.g., ${new Date().toISOString()})`,
              severity: 'warning'
            })
          }
        }
      }
    })
  }

  getType(value) {
    if (value === null) return 'null'
    if (Array.isArray(value)) return 'array'
    return typeof value
  }

  /**
   * Get validation summary as text
   * @param {Object} validationResult
   * @returns {string}
   */
  getValidationSummary(validationResult) {
    const { valid, errors, warnings } = validationResult

    if (valid && errors.length === 0 && warnings.length === 0) {
      return '✅ Message is valid'
    }

    let summary = ''

    if (errors.length > 0) {
      summary += `❌ ${errors.length} error${errors.length > 1 ? 's' : ''}:\n`
      errors.forEach((err, i) => {
        summary += `  ${i + 1}. ${err.field}: ${err.message}\n`
      })
    }

    if (warnings.length > 0) {
      if (summary) summary += '\n'
      summary += `⚠️  ${warnings.length} warning${warnings.length > 1 ? 's' : ''}:\n`
      warnings.forEach((warn, i) => {
        summary += `  ${i + 1}. ${warn.field}: ${warn.message}\n`
      })
    }

    return summary.trim()
  }
}

export const ocppValidator = new OCPPValidator(ValidationMode.STRICT)
