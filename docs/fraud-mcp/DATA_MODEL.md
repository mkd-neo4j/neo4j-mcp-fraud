# Neo4j Fraud Detection Data Model

This document defines the Neo4j graph schema used by the fraud detection MCP tools. It is based on the official Neo4j Transaction & Account Data Model.

## Base Model References

This data model extends the official Neo4j data models:
- **Transaction & Account Base Model**: https://neo4j.com/developer/industry-use-cases/_attachments/transaction-base-model.txt
- **Fraud Event Sequence Model**: https://neo4j.com/developer/industry-use-cases/_attachments/fraud-event-sequence-model.txt

## Core Node Types (from Official Neo4j Data Models)

### Customer
```cypher
(:Customer {
  customerId: string,           // Unique customer identifier
  firstName: string,            // Customer's given name
  middleName: string,           // Optional: Middle name(s) or initial(s)
  lastName: string,             // Customer's family name
  dateOfBirth: date,            // Date of birth for identity verification
  placeOfBirth: string,         // City where customer was born
  countryOfBirth: string        // ISO 3166-1 country code of birth
})
```

**Proposed Fraud-Specific Properties** (to be added):
```cypher
// These properties should be added to support fraud detection tools
riskScore: float,               // Current calculated risk score (0-10)
isPEP: boolean,                 // Politically Exposed Person flag
isFraudster: boolean,           // Known fraudster flag (confirmed fraud)
isSanctioned: boolean,          // On sanctions list
lastRiskAssessment: datetime    // When risk score was last calculated
```

### Account
```cypher
(:Account {
  accountNumber: string,        // Unique account identifier (IBAN, etc.)
  accountType: string,          // "CURRENT", "SAVINGS", "BUSINESS", "LOAN"
  openedDate: datetime,         // When account was opened
  closedDate: datetime,         // When account was closed (null if active)
  suspendedDate: datetime       // When account was suspended (null if not)
})
```

**Labels**:
- `:Internal` - Accounts held within this bank
- `:External` - Accounts at other financial institutions
- `:HighRiskJurisdiction` - Accounts in high-risk countries
- `:Flagged` _(Proposed)_ - Accounts flagged by fraud detection
- `:UnderInvestigation` _(Proposed)_ - Accounts under investigation
- `:Confirmed` _(Proposed)_ - Confirmed fraudulent accounts

### Transaction
```cypher
(:Transaction {
  transactionId: string,        // Unique transaction identifier
  amount: float,                // Monetary value (always positive)
  currency: string,             // ISO 4217 currency code (GBP, USD, EUR)
  date: datetime,               // When transaction was processed
  message: string,              // Payment reference/description
  type: string                  // "SWIFT", "ACH", "FASTER_PAYMENT", "CARD"
})
```

### Movement
```cypher
(:Movement {
  movementId: string,           // Unique movement identifier
  amount: float,                // Monetary value of this movement
  currency: string,             // ISO 4217 currency code
  date: datetime,               // When movement was executed
  description: string,          // Human-readable description
  status: string,               // "PENDING", "COMPLETED", "CANCELLED", "FAILED"
  sequenceNumber: integer,      // Order within series (starts from 1)
  authorisedBy: string,         // Who authorized this movement
  validatedBy: string,          // Secondary approval (dual control)
  createdAt: datetime           // When movement was created
})
```

### Device
```cypher
(:Device {
  deviceId: string,             // Unique device fingerprint
  deviceType: string,           // "mobile", "desktop", "tablet", "unknown"
  userAgent: string,            // Browser/app user agent string
  createdAt: datetime           // When device was first detected
})
```

### IP
```cypher
(:IP {
  ipAddress: string,            // IPv4 or IPv6 address
  createdAt: datetime           // When IP was first observed
})
```

### Session
```cypher
(:Session {
  sessionId: string,            // Unique session identifier
  status: string,               // "success", "failed", "suspicious", "timeout"
  createdAt: datetime           // When session was initiated
})
```

### Address
```cypher
(:Address {
  addressLine1: string,         // House/building number and street
  addressLine2: string,         // Optional: Flat, building name
  postTown: string,             // Town/city for postal delivery
  postCode: string,             // Postal code
  region: string,               // County, state, or region
  latitude: float,              // Geographic latitude
  longitude: float,             // Geographic longitude
  createdAt: datetime           // When address was recorded
})
```

**Proposed Fraud Property**:
```cypher
isHighRisk: boolean             // High-risk jurisdiction flag
```

### Email
```cypher
(:Email {
  address: string,              // Complete email address
  domain: string,               // Domain portion (e.g., "example.com")
  createdAt: datetime           // When email was recorded
})
```

### Phone
```cypher
(:Phone {
  number: string,               // Complete phone number with country code
  countryCode: string,          // International code (e.g., "+44", "+1")
  createdAt: datetime           // When phone was recorded
})
```

### Passport
```cypher
(:Passport {
  passportNumber: string,       // Passport number
  issueDate: date,              // When passport was issued
  expiryDate: date,             // When passport expires
  issuingCountry: string,       // ISO 3166-1 country code
  nationality: string,          // Nationality on passport
  createdAt: datetime           // When record was created
})
```

### DrivingLicense
```cypher
(:DrivingLicense {
  licenseNumber: string,        // License number
  issueDate: date,              // When license was issued
  expiryDate: date,             // When license expires
  issuingCountry: string,       // ISO 3166-1 country code
  createdAt: datetime           // When record was created
})
```

### Country
```cypher
(:Country {
  code: string,                 // ISO 3166-1 alpha-2 code (e.g., "GB", "US")
  name: string                  // Full country name
})
```

**Proposed Fraud Property**:
```cypher
isHighRisk: boolean,            // FATF blacklist or sanctions
riskLevel: string               // "low", "medium", "high", "critical"
```

### Counterparty
```cypher
(:Counterparty {
  counterpartyId: string,       // Unique counterparty identifier
  name: string,                 // Legal name
  type: string,                 // "INDIVIDUAL", "BUSINESS", "GOVERNMENT", "CHARITY"
  registrationNumber: string,   // Official registration number
  createdAt: datetime           // When counterparty was recorded
})
```

### Location
```cypher
(:Location {
  city: string,                 // City name
  postCode: string,             // Postal code (may be partial)
  country: string,              // ISO 3166-1 country code
  latitude: float,              // Geographic latitude
  longitude: float,             // Geographic longitude
  createdAt: datetime           // When location was recorded
})
```

### ISP
```cypher
(:ISP {
  name: string,                 // Internet Service Provider name
  createdAt: datetime           // When ISP was recorded
})
```

### Alert _(Proposed)_
```cypher
(:Alert {
  alertId: string,              // Unique alert identifier
  ruleName: string,             // Fraud rule that triggered alert
  ruleId: string,               // System identifier for rule
  severity: string,             // "LOW", "MEDIUM", "HIGH", "CRITICAL"
  triggeredAt: datetime         // When alert was triggered
})
```

### Case _(Proposed)_
```cypher
(:Case {
  caseId: string,               // Unique case identifier
  status: string,               // "OPEN", "UNDER_INVESTIGATION", "CLOSED", "ESCALATED"
  outcome: string,              // "PROVEN_FRAUD", "NOT_FRAUD", etc.
  financialStakes: float,       // Monetary value at risk
  investigatedBy: string,       // Investigator user ID
  createdAt: datetime,          // When case was opened
  closedAt: datetime            // When case was closed (null if open)
})
```

## Fraud Event Sequence Nodes (from Official Model)

For account takeover detection:

### Authentication
```cypher
(:Authentication {
  method: string,               // "email", "phone_number", "biometric"
  status: string,               // "success", "failed"
  createdAt: datetime           // When authentication occurred
})
```

### ChangePhone
```cypher
(:ChangePhone {
  createdAt: datetime           // When phone was changed
})
```

### ChangeEmail
```cypher
(:ChangeEmail {
  createdAt: datetime           // When email was changed
})
```

### ChangeAddress
```cypher
(:ChangeAddress {
  createdAt: datetime           // When address was changed
})
```

### AddExternalAccount
```cypher
(:AddExternalAccount {
  createdAt: datetime           // When external account was added
})
```

### Transfer
```cypher
(:Transfer {
  createdAt: datetime           // When transfer was initiated
})
```

## Core Relationship Types (from Official Neo4j Data Models)

```cypher
// Customer identity relationships
(:Customer)-[:HAS_ACCOUNT {role: string, since: datetime}]->(:Account)
(:Customer)-[:HAS_ADDRESS {addedAt: datetime, lastChangedAt: datetime, isCurrent: boolean}]->(:Address)
(:Customer)-[:HAS_EMAIL {since: datetime}]->(:Email)
(:Customer)-[:HAS_PHONE {since: datetime}]->(:Phone)
(:Customer)-[:HAS_PASSPORT {verificationDate: datetime, verificationMethod: string, verificationStatus: string}]->(:Passport)
(:Customer)-[:HAS_DRIVING_LICENSE {verificationDate: datetime, verificationMethod: string, verificationStatus: string}]->(:DrivingLicense)
(:Customer)-[:HAS_NATIONALITY]->(:Country)

// Transaction flow relationships
(:Account)-[:PERFORMS]->(:Transaction)
(:Transaction)-[:BENEFITS_TO]->(:Account)
(:Transaction)-[:IMPLIED {totalMovements: integer}]->(:Movement)

// Account location
(:Account)-[:IS_HOSTED]->(:Country)

// Counterparty relationships
(:Counterparty)-[:HAS_ACCOUNT {since: datetime}]->(:Account)
(:Counterparty)-[:HAS_ADDRESS {since: datetime, isCurrent: boolean}]->(:Address)

// Session and device relationships
(:Session)-[:USES_IP]->(:IP)
(:Session)-[:SESSION_USES_DEVICE]->(:Device)
(:Device)-[:USED_BY {lastUsed: datetime}]->(:Customer)

// IP geolocation
(:IP)-[:IS_ALLOCATED_TO {createdAt: datetime}]->(:ISP)
(:IP)-[:LOCATED_IN {createdAt: datetime}]->(:Location)

// Location hierarchy
(:Location)-[:LOCATED_IN]->(:Country)
(:Address)-[:LOCATED_IN]->(:Country)

// Fraud investigation relationships (proposed)
(:Account)-[:SUBJECT_OF]->(:Case)
(:Customer)-[:SUBJECT_OF]->(:Case)
(:Alert)-[:TRIGGERED]->(:Case)

// Event sequence relationships (for account takeover detection)
(:Customer)-[:CONNECTS]->(:Authentication)
(:Session)-[:HAS_AUTHENTICATION]->(:Authentication)
(:Session)-[:HAS_CHANGE_PHONE]->(:ChangePhone)
(:Session)-[:HAS_CHANGE_EMAIL]->(:ChangeEmail)
(:Session)-[:HAS_CHANGE_ADDRESS]->(:ChangeAddress)
(:Session)-[:HAS_ADD_EXTERNAL_ACCOUNT]->(:AddExternalAccount)
(:Session)-[:HAS_TRANSFER]->(:Transfer)

// Event chaining (chronological order)
(:Event)-[:NEXT]->(:Event)

// Event details
(:ChangePhone)-[:OLD_PHONE]->(:Phone)
(:ChangePhone)-[:NEW_PHONE]->(:Phone)
(:ChangeEmail)-[:OLD_EMAIL]->(:Email)
(:ChangeEmail)-[:NEW_EMAIL]->(:Email)
(:ChangeAddress)-[:OLD_ADDRESS]->(:Address)
(:ChangeAddress)-[:NEW_ADDRESS]->(:Address)
(:AddExternalAccount)-[:ADD_ACCOUNT]->(:Account)
(:Transfer)-[:HAS_TRANSACTION]->(:Transaction)
```

## Constraints and Indexes

For optimal fraud detection query performance, the following constraints and indexes should be created:

```cypher
// Node uniqueness constraints (from base model)
CREATE CONSTRAINT customer_id IF NOT EXISTS
FOR (c:Customer) REQUIRE c.customerId IS NODE KEY;

CREATE CONSTRAINT email_address IF NOT EXISTS
FOR (e:Email) REQUIRE e.address IS NODE KEY;

CREATE CONSTRAINT phone_number IF NOT EXISTS
FOR (p:Phone) REQUIRE p.number IS NODE KEY;

CREATE CONSTRAINT passport_number IF NOT EXISTS
FOR (p:Passport) REQUIRE (p.passportNumber, p.issuingCountry) IS NODE KEY;

CREATE CONSTRAINT driving_licence_number IF NOT EXISTS
FOR (d:DrivingLicense) REQUIRE (d.licenseNumber, d.issuingCountry) IS NODE KEY;

CREATE CONSTRAINT device_id IF NOT EXISTS
FOR (d:Device) REQUIRE d.deviceId IS NODE KEY;

CREATE CONSTRAINT ip_address IF NOT EXISTS
FOR (i:IP) REQUIRE i.ipAddress IS NODE KEY;

CREATE CONSTRAINT session_id IF NOT EXISTS
FOR (s:Session) REQUIRE s.sessionId IS NODE KEY;

CREATE CONSTRAINT account_number IF NOT EXISTS
FOR (a:Account) REQUIRE a.accountNumber IS NODE KEY;

CREATE CONSTRAINT transaction_id IF NOT EXISTS
FOR (t:Transaction) REQUIRE t.transactionId IS NODE KEY;

CREATE CONSTRAINT counterparty_id IF NOT EXISTS
FOR (cp:Counterparty) REQUIRE cp.counterpartyId IS NODE KEY;

CREATE CONSTRAINT movement_id IF NOT EXISTS
FOR (m:Movement) REQUIRE m.movementId IS NODE KEY;

CREATE CONSTRAINT isp_name IF NOT EXISTS
FOR (i:ISP) REQUIRE i.name IS NODE KEY;

CREATE CONSTRAINT country_code IF NOT EXISTS
FOR (c:Country) REQUIRE c.code IS NODE KEY;

CREATE CONSTRAINT address_composite IF NOT EXISTS
FOR (a:Address) REQUIRE (a.addressLine1, a.postTown, a.postCode) IS NODE KEY;

// Proposed: Fraud investigation constraints
CREATE CONSTRAINT alert_id IF NOT EXISTS
FOR (a:Alert) REQUIRE a.alertId IS NODE KEY;

CREATE CONSTRAINT case_id IF NOT EXISTS
FOR (c:Case) REQUIRE c.caseId IS NODE KEY;

// Performance indexes for fraud detection
CREATE INDEX transaction_date_idx IF NOT EXISTS
FOR (t:Transaction) ON (t.date);

CREATE INDEX transaction_amount_idx IF NOT EXISTS
FOR (t:Transaction) ON (t.amount);

CREATE INDEX device_type_idx IF NOT EXISTS
FOR (d:Device) ON (d.deviceType);

CREATE INDEX session_status_idx IF NOT EXISTS
FOR (s:Session) ON (s.status);

// Proposed: Fraud-specific indexes
CREATE INDEX customer_risk_score_idx IF NOT EXISTS
FOR (c:Customer) ON (c.riskScore);

CREATE INDEX customer_pep_idx IF NOT EXISTS
FOR (c:Customer) ON (c.isPEP);

CREATE INDEX customer_fraudster_idx IF NOT EXISTS
FOR (c:Customer) ON (c.isFraudster);

CREATE INDEX address_highrisk_idx IF NOT EXISTS
FOR (a:Address) ON (a.isHighRisk);

CREATE INDEX country_highrisk_idx IF NOT EXISTS
FOR (c:Country) ON (c.isHighRisk);
```

## Fraud-Specific Data Model Extensions

### Required Properties for Fraud Tools

To fully support the fraud detection tools, add these properties to existing nodes:

**Customer Node Extensions**:
```cypher
MATCH (c:Customer)
SET c.riskScore = coalesce(c.riskScore, 5.0),
    c.isPEP = coalesce(c.isPEP, false),
    c.isFraudster = coalesce(c.isFraudster, false),
    c.isSanctioned = coalesce(c.isSanctioned, false),
    c.lastRiskAssessment = coalesce(c.lastRiskAssessment, datetime())
```

**Address Node Extensions**:
```cypher
MATCH (a:Address)
SET a.isHighRisk = coalesce(a.isHighRisk, false)
```

**Country Node Extensions**:
```cypher
// Mark known high-risk jurisdictions (FATF list example)
MATCH (c:Country)
WHERE c.code IN ['KP', 'IR', 'MM', 'SY']  // North Korea, Iran, Myanmar, Syria
SET c.isHighRisk = true,
    c.riskLevel = 'critical'
```

### Migration Guide

For existing Neo4j databases without fraud-specific properties:

```cypher
// Step 1: Add fraud properties to Customer nodes
MATCH (c:Customer)
SET c.isFraudster = coalesce(c.isFraudster, false),
    c.isPEP = coalesce(c.isPEP, false),
    c.isSanctioned = coalesce(c.isSanctioned, false),
    c.riskScore = coalesce(c.riskScore, 5.0);

// Step 2: Add high-risk flags to Address nodes
MATCH (a:Address)
SET a.isHighRisk = coalesce(a.isHighRisk, false);

// Step 3: Mark high-risk jurisdictions
MATCH (a:Address)-[:LOCATED_IN]->(c:Country)
WHERE c.code IN ['KP', 'IR', 'SY', 'VE', 'ZW']  // Example FATF high-risk
SET a.isHighRisk = true;

MATCH (c:Country)
WHERE c.code IN ['KP', 'IR', 'SY', 'VE', 'ZW']
SET c.isHighRisk = true,
    c.riskLevel = 'critical';

// Step 4: Add HighRiskJurisdiction label to relevant accounts
MATCH (a:Account)-[:IS_HOSTED]->(c:Country {isHighRisk: true})
SET a:HighRiskJurisdiction;
```

## Data Model Best Practices

When extending or implementing this data model:

1. **Follow Neo4j naming conventions**:
   - Node labels: CamelCase (e.g., `Customer`, `DrivingLicense`)
   - Relationship types: ALL_CAPS with underscores (e.g., `HAS_ACCOUNT`, `PERFORMS`)
   - Properties: camelCase (e.g., `customerId`, `dateOfBirth`)

2. **Use appropriate data types**:
   - Dates: Use `date()` type for dates without time
   - Timestamps: Use `datetime()` for full timestamps
   - Currency: Use `float` for amounts, always store in smallest unit if needed

3. **Maintain data integrity**:
   - Always use NODE KEY constraints for unique identifiers
   - Composite keys where necessary (e.g., passport + issuing country)
   - Validate relationship direction matches official model

4. **Optimize for graph traversals**:
   - Index properties used in WHERE clauses
   - Index properties used for range queries (dates, amounts)
   - Consider relationship indexes for high-cardinality paths

5. **Keep relationship metadata minimal**:
   - Only add properties that are truly relationship-specific
   - Move entity properties to nodes, not relationships
   - Use relationship types to encode semantics where possible

## References

- **Neo4j Data Model Best Practices**: https://neo4j.com/developer/industry-use-cases/_attachments/neo4j_data_model_best_practices.txt
- **Transaction Base Model**: https://neo4j.com/developer/industry-use-cases/_attachments/transaction-base-model.txt
- **Fraud Event Sequence Model**: https://neo4j.com/developer/industry-use-cases/_attachments/fraud-event-sequence-model.txt
