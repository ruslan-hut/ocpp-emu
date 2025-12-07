# OCPP Emulator - Testing Checklist
**Current Stage**: End of Phase 2 + Phase 3 Frontend Features  
**Date**: 2025-01-06  
**Purpose**: Verify all implemented features before continuing development

---

## 🎯 Critical Path Tests (Must Pass)

### 1. Application Startup & Infrastructure
- [ ] **1.1** Application starts without errors
- [ ] **1.2** MongoDB connection established successfully
- [ ] **1.3** MongoDB collections created (stations, messages, transactions, sessions)
- [ ] **1.4** MongoDB indexes created and verified
- [ ] **1.5** HTTP server starts on configured port (default: 8080)
- [ ] **1.6** WebSocket server for frontend communication starts
- [ ] **1.7** Frontend loads and displays without errors
- [ ] **1.8** Frontend connects to backend WebSocket successfully
- [ ] **1.9** Configuration file (config.yaml) loads correctly
- [ ] **1.10** Station Manager loads stations from MongoDB on startup

### 2. Station Management (CRUD Operations)
- [ ] **2.1** Create new station via Web UI
  - [ ] All required fields validated
  - [ ] Station saved to MongoDB
  - [ ] Station appears in station list
- [ ] **2.2** View station details
  - [ ] All station properties displayed correctly
  - [ ] Connector information shown
  - [ ] Runtime state visible
- [ ] **2.3** Edit existing station
  - [ ] Changes saved to MongoDB
  - [ ] Changes reflected in UI immediately
  - [ ] No restart required
- [ ] **2.4** Delete station
  - [ ] Station removed from MongoDB
  - [ ] Station removed from UI
  - [ ] If connected, connection terminated
- [ ] **2.5** Station form validation
  - [ ] Required fields enforced
  - [ ] Invalid data rejected
  - [ ] Error messages displayed

### 3. Station Connection & OCPP Protocol
- [ ] **3.1** Start station connection
  - [ ] WebSocket connection to CSMS established
  - [ ] BootNotification sent automatically
  - [ ] Connection status updated in UI
  - [ ] Connection status synced to MongoDB
- [ ] **3.2** Stop station connection
  - [ ] WebSocket connection closed gracefully
  - [ ] Connection status updated
  - [ ] No errors in logs
- [ ] **3.3** Auto-start functionality
  - [ ] Stations with `auto_start: true` connect on app startup
  - [ ] Multiple stations can auto-start simultaneously
- [ ] **3.4** BootNotification message
  - [ ] Sent on connection
  - [ ] Contains correct station information (vendor, model, serial)
  - [ ] Response received and logged
  - [ ] Registration status handled correctly
- [ ] **3.5** Heartbeat messages
  - [ ] Sent at configured interval (default: 60s)
  - [ ] Response received and logged
  - [ ] Heartbeat indicator updates in UI
  - [ ] Last heartbeat timestamp synced to MongoDB
- [ ] **3.6** StatusNotification messages
  - [ ] Sent when connector status changes
  - [ ] All connector states supported (Available, Occupied, Faulted, etc.)
  - [ ] Status changes reflected in UI
- [ ] **3.7** Multiple station connections
  - [ ] Can connect 2-5 stations simultaneously
  - [ ] Each station maintains independent state
  - [ ] No cross-station interference
  - [ ] All connections stable

### 4. Charging Session Flow (Complete End-to-End)
- [ ] **4.1** Start charging session
  - [ ] Authorize request sent with idTag
  - [ ] Authorization response received
  - [ ] StartTransaction sent
  - [ ] Transaction ID assigned
  - [ ] Connector status changes to "Charging"
  - [ ] Transaction saved to MongoDB
- [ ] **4.2** Meter values during charging
  - [ ] MeterValues sent at configured interval
  - [ ] Energy values increment correctly
  - [ ] Power values realistic
  - [ ] Meter values logged to MongoDB
  - [ ] Meter values visible in UI
- [ ] **4.3** Stop charging session
  - [ ] StopTransaction sent
  - [ ] Final meter value included
  - [ ] Transaction completed in MongoDB
  - [ ] Connector status returns to "Available"
  - [ ] Energy consumption calculated correctly
- [ ] **4.4** Complete session verification
  - [ ] All messages logged (Authorize, StartTransaction, MeterValues, StopTransaction)
  - [ ] Transaction record complete in MongoDB
  - [ ] State machine transitions correct
  - [ ] No errors in logs

### 5. Message Logging & Persistence
- [ ] **5.1** All messages logged to MongoDB
  - [ ] Sent messages logged
  - [ ] Received messages logged
  - [ ] Message metadata correct (station ID, direction, type, timestamp)
  - [ ] Payload stored correctly
- [ ] **5.2** Message correlation
  - [ ] Request/response messages linked by message ID
  - [ ] Correlation ID works correctly
- [ ] **5.3** Message buffer and batching
  - [ ] Messages buffered in memory
  - [ ] Batch inserts to MongoDB work
  - [ ] No message loss during high load
  - [ ] Buffer overflow handled gracefully
- [ ] **5.4** Transaction persistence
  - [ ] Transactions saved to MongoDB
  - [ ] Transaction fields complete (ID, station, connector, idTag, timestamps, meter values)
  - [ ] Transaction status updates correctly

### 6. Real-Time Message Streaming
- [ ] **6.1** WebSocket connection to frontend
  - [ ] Connection established successfully
  - [ ] Connection status indicator shows "connected"
  - [ ] Reconnection works on disconnect
- [ ] **6.2** Real-time message updates
  - [ ] New messages appear in UI immediately
  - [ ] No page refresh required
  - [ ] Message direction indicators correct (sent/received)
  - [ ] Timestamps displayed correctly
- [ ] **6.3** Live updates toggle
  - [ ] Can enable/disable live updates
  - [ ] When disabled, messages still load from API
  - [ ] Visual indicator shows live updates status
- [ ] **6.4** Message filtering in real-time
  - [ ] Filter by station ID works
  - [ ] Filter by direction works
  - [ ] Filter by message type works
  - [ ] Filters apply to new messages

### 7. Message Inspector UI
- [ ] **7.1** Message list display
  - [ ] Messages displayed in chronological order
  - [ ] Message details visible (station, type, action, timestamp)
  - [ ] JSON payload formatted correctly
  - [ ] Syntax highlighting works (if implemented)
- [ ] **7.2** Message filtering
  - [ ] Filter by station ID
  - [ ] Filter by direction (sent/received)
  - [ ] Filter by message type (Call/CallResult/CallError)
  - [ ] Filter by action name
  - [ ] Search by content
  - [ ] Multiple filters work together
- [ ] **7.3** Message details
  - [ ] Click message to view full details
  - [ ] Payload displayed in readable format
  - [ ] Request/response correlation visible
  - [ ] Error details shown for CallError messages
- [ ] **7.4** Message export
  - [ ] Export to JSON works
  - [ ] Export to CSV works
  - [ ] Exported file contains all visible messages
  - [ ] File downloads correctly
  - [ ] File format is valid

### 8. Custom Message Crafter
- [ ] **8.1** Message Crafter UI loads
  - [ ] Station selector works
  - [ ] Message type selector (Call/CallResult/CallError)
  - [ ] Action selector for Call messages
  - [ ] JSON editor (Monaco) loads
- [ ] **8.2** JSON editor functionality
  - [ ] Syntax highlighting works
  - [ ] JSON validation works
  - [ ] Error indicators show invalid JSON
  - [ ] Auto-format works
- [ ] **8.3** Message templates
  - [ ] Template library accessible
  - [ ] Can select template
  - [ ] Template populates editor
  - [ ] Can save custom templates
  - [ ] Can delete templates
- [ ] **8.4** Message validation
  - [ ] Validation can be enabled/disabled
  - [ ] Validation mode selection (strict/lenient)
  - [ ] Validation errors displayed
  - [ ] Can send invalid messages when validation disabled
- [ ] **8.5** Send custom message
  - [ ] Message sent to selected station
  - [ ] Message appears in message inspector
  - [ ] Response received and displayed
  - [ ] Error handling for failed sends

### 9. Station Templates & Import/Export
- [ ] **9.1** Station templates
  - [ ] Templates manager accessible
  - [ ] Default templates available
  - [ ] Can create new template from station
  - [ ] Can create station from template
  - [ ] Can edit templates
  - [ ] Can delete templates
  - [ ] Templates persist in localStorage
- [ ] **9.2** Import stations
  - [ ] Import from JSON file works
  - [ ] Valid JSON file accepted
  - [ ] Invalid JSON rejected with error
  - [ ] Imported stations appear in list
  - [ ] Imported stations can be started
- [ ] **9.3** Export stations
  - [ ] Export all stations to JSON
  - [ ] Export single station to JSON
  - [ ] Exported file is valid JSON
  - [ ] Exported file can be re-imported
  - [ ] File downloads correctly

### 10. Configuration Management UI
- [ ] **10.1** Configuration panel accessible
  - [ ] Application settings displayed (read-only)
  - [ ] MongoDB connection status shown
  - [ ] Server info displayed
  - [ ] CSMS default settings shown
- [ ] **10.2** System status
  - [ ] MongoDB health check displayed
  - [ ] Active WebSocket connections count
  - [ ] Connection statistics visible
- [ ] **10.3** Station configuration editor
  - [ ] Can edit station configuration
  - [ ] All configuration sections accessible
  - [ ] Changes saved correctly

### 11. Dashboard
- [ ] **11.1** Dashboard displays correctly
  - [ ] All active stations shown
  - [ ] Connection status indicators accurate
  - [ ] Active transactions count correct
  - [ ] Recent message activity visible
- [ ] **11.2** Quick actions
  - [ ] Start station from dashboard
  - [ ] Stop station from dashboard
  - [ ] Actions update UI immediately

### 12. Error Handling & Edge Cases
- [ ] **12.1** Connection failures
  - [ ] Invalid CSMS URL handled gracefully
  - [ ] Connection timeout handled
  - [ ] Network errors don't crash application
  - [ ] Error messages displayed to user
- [ ] **12.2** MongoDB failures
  - [ ] Application handles MongoDB disconnection
  - [ ] Error logged but app continues
  - [ ] Reconnection attempted
- [ ] **12.3** Invalid messages
  - [ ] Malformed OCPP messages handled
  - [ ] CallError responses handled
  - [ ] Invalid JSON in custom messages handled
- [ ] **12.4** Concurrent operations
  - [ ] Multiple stations can start simultaneously
  - [ ] Multiple charging sessions can run concurrently
  - [ ] No race conditions in state updates

### 13. State Persistence
- [ ] **13.1** Application restart
  - [ ] Stations persist across restart
  - [ ] Station configurations loaded from MongoDB
  - [ ] Auto-start stations reconnect
  - [ ] Message history preserved
  - [ ] Transaction history preserved
- [ ] **13.2** Runtime state sync
  - [ ] Connection status synced to MongoDB
  - [ ] Last heartbeat synced
  - [ ] Connector status synced
  - [ ] Transaction state synced

### 14. Performance & Scalability
- [ ] **14.1** Multiple stations
  - [ ] 5 stations can run simultaneously
  - [ ] No performance degradation
  - [ ] Memory usage reasonable
  - [ ] CPU usage reasonable
- [ ] **14.2** Message throughput
  - [ ] High message rate handled (100+ messages/min)
  - [ ] No message loss
  - [ ] UI remains responsive
- [ ] **14.3** Database performance
  - [ ] MongoDB queries perform well
  - [ ] Indexes used effectively
  - [ ] Batch inserts work efficiently

---

## 🔍 Integration Tests

### 15. End-to-End Charging Session
- [ ] **15.1** Complete session with real CSMS (if available)
  - [ ] Connect to external CSMS
  - [ ] Complete full charging session
  - [ ] Verify all messages exchanged correctly
  - [ ] Verify transaction recorded
- [ ] **15.2** Multiple concurrent sessions
  - [ ] 2-3 stations charging simultaneously
  - [ ] All sessions complete successfully
  - [ ] No interference between sessions

### 16. Docker Deployment
- [ ] **16.1** Docker Compose setup
  - [ ] All services start correctly
  - [ ] MongoDB initializes
  - [ ] Backend connects to MongoDB
  - [ ] Frontend connects to backend
- [ ] **16.2** Container health
  - [ ] Health checks pass
  - [ ] Containers restart on failure
  - [ ] Logs accessible

---

## 📊 Test Results Summary

### Test Execution
- **Date**: _______________
- **Tester**: _______________
- **Environment**: _______________
- **MongoDB Version**: _______________
- **Go Version**: _______________
- **Node Version**: _______________

### Results
- **Total Tests**: ___
- **Passed**: ___
- **Failed**: ___
- **Blocked**: ___
- **Pass Rate**: ___%

### Critical Issues Found
1. _________________________________________________
2. _________________________________________________
3. _________________________________________________

### Non-Critical Issues Found
1. _________________________________________________
2. _________________________________________________
3. _________________________________________________

### Recommendations
- [ ] **Ready to continue development** - All critical tests pass
- [ ] **Fix critical issues first** - Blocking issues must be resolved
- [ ] **Address non-critical issues** - Can continue but should fix soon
- [ ] **Performance concerns** - May need optimization before scaling

---

## 🚨 Blocking Issues (Must Fix Before Continuing)

If any of these fail, development should be paused:

1. **Application startup** (Section 1)
2. **Station CRUD operations** (Section 2)
3. **Station connection** (Section 3)
4. **Complete charging session** (Section 4)
5. **Message persistence** (Section 5)
6. **Real-time streaming** (Section 6)

---

## 📝 Notes

_Use this section for additional observations, suggestions, or issues discovered during testing:_




