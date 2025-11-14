// Template storage service using localStorage

const STORAGE_KEY = 'ocpp_message_templates'

export const templateService = {
  // Get all custom templates
  getCustomTemplates() {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      return stored ? JSON.parse(stored) : []
    } catch (err) {
      console.error('Failed to load templates:', err)
      return []
    }
  },

  // Save a custom template
  saveTemplate(template) {
    try {
      const templates = this.getCustomTemplates()

      // Check if template with same name exists
      const existingIndex = templates.findIndex(t => t.name === template.name)

      if (existingIndex >= 0) {
        // Update existing template
        templates[existingIndex] = {
          ...template,
          updatedAt: new Date().toISOString()
        }
      } else {
        // Add new template
        templates.push({
          ...template,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString()
        })
      }

      localStorage.setItem(STORAGE_KEY, JSON.stringify(templates))
      return true
    } catch (err) {
      console.error('Failed to save template:', err)
      return false
    }
  },

  // Delete a custom template
  deleteTemplate(name) {
    try {
      const templates = this.getCustomTemplates()
      const filtered = templates.filter(t => t.name !== name)
      localStorage.setItem(STORAGE_KEY, JSON.stringify(filtered))
      return true
    } catch (err) {
      console.error('Failed to delete template:', err)
      return false
    }
  },

  // Get built-in templates
  getBuiltInTemplates() {
    return {
      // Core Profile
      'Heartbeat': {
        category: 'Core',
        description: 'Send heartbeat to keep connection alive',
        action: 'Heartbeat',
        payload: {}
      },
      'BootNotification': {
        category: 'Core',
        description: 'Notify CSMS of charge point boot',
        action: 'BootNotification',
        payload: {
          chargePointVendor: "VendorName",
          chargePointModel: "ModelX",
          chargePointSerialNumber: "SN123456",
          chargeBoxSerialNumber: "CB123456",
          firmwareVersion: "1.0.0",
          iccid: "89310410106543789301",
          imsi: "310410123456789",
          meterType: "EnergyMeter",
          meterSerialNumber: "MTR123456"
        }
      },
      'StatusNotification': {
        category: 'Core',
        description: 'Report connector status change',
        action: 'StatusNotification',
        payload: {
          connectorId: 1,
          errorCode: "NoError",
          status: "Available",
          timestamp: new Date().toISOString(),
          info: "",
          vendorId: "",
          vendorErrorCode: ""
        }
      },
      'Authorize': {
        category: 'Core',
        description: 'Request authorization for ID tag',
        action: 'Authorize',
        payload: {
          idTag: "TAG123456"
        }
      },
      'StartTransaction': {
        category: 'Core',
        description: 'Start a charging transaction',
        action: 'StartTransaction',
        payload: {
          connectorId: 1,
          idTag: "TAG123456",
          meterStart: 0,
          timestamp: new Date().toISOString(),
          reservationId: null
        }
      },
      'StopTransaction': {
        category: 'Core',
        description: 'Stop a charging transaction',
        action: 'StopTransaction',
        payload: {
          transactionId: 1,
          meterStop: 15000,
          timestamp: new Date().toISOString(),
          reason: "Local",
          idTag: "TAG123456",
          transactionData: []
        }
      },
      'MeterValues': {
        category: 'Core',
        description: 'Send meter values during transaction',
        action: 'MeterValues',
        payload: {
          connectorId: 1,
          transactionId: 1,
          meterValue: [{
            timestamp: new Date().toISOString(),
            sampledValue: [{
              value: "1000",
              context: "Sample.Periodic",
              format: "Raw",
              measurand: "Energy.Active.Import.Register",
              phase: null,
              location: "Outlet",
              unit: "Wh"
            }]
          }]
        }
      },
      'DataTransfer': {
        category: 'Core',
        description: 'Send vendor-specific data',
        action: 'DataTransfer',
        payload: {
          vendorId: "VendorName",
          messageId: "CustomMessage",
          data: "custom data here"
        }
      },

      // Firmware Management
      'DiagnosticsStatusNotification': {
        category: 'Firmware',
        description: 'Report diagnostics upload status',
        action: 'DiagnosticsStatusNotification',
        payload: {
          status: "Uploaded"
        }
      },
      'FirmwareStatusNotification': {
        category: 'Firmware',
        description: 'Report firmware update status',
        action: 'FirmwareStatusNotification',
        payload: {
          status: "Downloading"
        }
      },

      // Remote Trigger
      'TriggerMessage (Response)': {
        category: 'Remote',
        description: 'Response to TriggerMessage request',
        action: 'TriggerMessage',
        payload: {
          status: "Accepted"
        }
      },

      // Reservation
      'ReserveNow (Response)': {
        category: 'Reservation',
        description: 'Response to ReserveNow request',
        action: 'ReserveNow',
        payload: {
          status: "Accepted"
        }
      },
      'CancelReservation (Response)': {
        category: 'Reservation',
        description: 'Response to CancelReservation request',
        action: 'CancelReservation',
        payload: {
          status: "Accepted"
        }
      },

      // Smart Charging
      'SetChargingProfile (Response)': {
        category: 'SmartCharging',
        description: 'Response to SetChargingProfile request',
        action: 'SetChargingProfile',
        payload: {
          status: "Accepted"
        }
      },
      'ClearChargingProfile (Response)': {
        category: 'SmartCharging',
        description: 'Response to ClearChargingProfile request',
        action: 'ClearChargingProfile',
        payload: {
          status: "Accepted"
        }
      },
      'GetCompositeSchedule (Response)': {
        category: 'SmartCharging',
        description: 'Response to GetCompositeSchedule request',
        action: 'GetCompositeSchedule',
        payload: {
          status: "Accepted",
          connectorId: 1,
          scheduleStart: new Date().toISOString(),
          chargingSchedule: {
            duration: 3600,
            startSchedule: new Date().toISOString(),
            chargingRateUnit: "W",
            chargingSchedulePeriod: [{
              startPeriod: 0,
              limit: 22000
            }]
          }
        }
      },

      // Configuration
      'ChangeConfiguration (Response)': {
        category: 'Configuration',
        description: 'Response to ChangeConfiguration request',
        action: 'ChangeConfiguration',
        payload: {
          status: "Accepted"
        }
      },
      'GetConfiguration (Response)': {
        category: 'Configuration',
        description: 'Response to GetConfiguration request',
        action: 'GetConfiguration',
        payload: {
          configurationKey: [{
            key: "HeartbeatInterval",
            readonly: false,
            value: "60"
          }],
          unknownKey: []
        }
      },

      // Remote Control
      'RemoteStartTransaction (Response)': {
        category: 'Remote',
        description: 'Response to RemoteStartTransaction request',
        action: 'RemoteStartTransaction',
        payload: {
          status: "Accepted"
        }
      },
      'RemoteStopTransaction (Response)': {
        category: 'Remote',
        description: 'Response to RemoteStopTransaction request',
        action: 'RemoteStopTransaction',
        payload: {
          status: "Accepted"
        }
      },
      'UnlockConnector (Response)': {
        category: 'Remote',
        description: 'Response to UnlockConnector request',
        action: 'UnlockConnector',
        payload: {
          status: "Unlocked"
        }
      },
      'Reset (Response)': {
        category: 'Remote',
        description: 'Response to Reset request',
        action: 'Reset',
        payload: {
          status: "Accepted"
        }
      },
      'ChangeAvailability (Response)': {
        category: 'Remote',
        description: 'Response to ChangeAvailability request',
        action: 'ChangeAvailability',
        payload: {
          status: "Accepted"
        }
      }
    }
  },

  // Get all templates (built-in + custom)
  getAllTemplates() {
    const builtIn = this.getBuiltInTemplates()
    const custom = this.getCustomTemplates()

    // Convert built-in templates to array format
    const builtInArray = Object.entries(builtIn).map(([name, template]) => ({
      name,
      ...template,
      isBuiltIn: true
    }))

    // Mark custom templates
    const customArray = custom.map(t => ({
      ...t,
      isBuiltIn: false,
      category: t.category || 'Custom'
    }))

    return [...builtInArray, ...customArray]
  },

  // Get templates by category
  getTemplatesByCategory() {
    const allTemplates = this.getAllTemplates()
    const categories = {}

    allTemplates.forEach(template => {
      const category = template.category || 'Other'
      if (!categories[category]) {
        categories[category] = []
      }
      categories[category].push(template)
    })

    return categories
  }
}
