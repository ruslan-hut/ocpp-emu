// MongoDB initialization script
// This script runs automatically when MongoDB container starts for the first time

db = db.getSiblingDB('ocpp_emu');

print('Creating collections...');

// Create messages collection
db.createCollection('messages');
print('Created messages collection');

// Create transactions collection
db.createCollection('transactions');
print('Created transactions collection');

// Create stations collection
db.createCollection('stations');
print('Created stations collection');

// Create sessions collection
db.createCollection('sessions');
print('Created sessions collection');

// Create time-series collection for meter values
db.createCollection('meter_values', {
  timeseries: {
    timeField: 'timestamp',
    metaField: 'metadata',
    granularity: 'seconds'
  }
});
print('Created meter_values time-series collection');

print('Creating indexes...');

// Indexes for messages collection
db.messages.createIndex({ station_id: 1, timestamp: -1 });
db.messages.createIndex({ message_id: 1 });
db.messages.createIndex({ correlation_id: 1 });
db.messages.createIndex({ action: 1, timestamp: -1 });
db.messages.createIndex({ timestamp: -1 });
// Optional: TTL index to auto-delete old messages after 30 days
// db.messages.createIndex({ created_at: 1 }, { expireAfterSeconds: 2592000 });
print('Created indexes for messages');

// Indexes for transactions collection
db.transactions.createIndex({ transaction_id: 1 }, { unique: true });
db.transactions.createIndex({ station_id: 1, start_timestamp: -1 });
db.transactions.createIndex({ status: 1 });
db.transactions.createIndex({ id_tag: 1 });
print('Created indexes for transactions');

// Indexes for stations collection
db.stations.createIndex({ station_id: 1 }, { unique: true });
db.stations.createIndex({ connection_status: 1 });
db.stations.createIndex({ enabled: 1, auto_start: 1 });
db.stations.createIndex({ tags: 1 });
db.stations.createIndex({ protocol_version: 1 });
print('Created indexes for stations');

// Indexes for sessions collection
db.sessions.createIndex({ station_id: 1, status: 1 });
db.sessions.createIndex({ status: 1 });
print('Created indexes for sessions');

// Text search indexes for message debugging
db.messages.createIndex({
  action: 'text',
  error_description: 'text'
});
print('Created text search index for messages');

print('MongoDB initialization complete!');
print('Database: ocpp_emu');
print('Collections created: messages, transactions, stations, sessions, meter_values');
print('All indexes created successfully');
