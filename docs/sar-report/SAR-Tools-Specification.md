# SAR Tools Specification
## Required Tools for Creating Complete Suspicious Activity Reports

This document maps the [SAR Requirements](SAR-Requirements.md) to the tools needed to gather evidence and create complete SAR filings from Neo4j fraud detection data.

---

## Executive Summary

### Current State
**Existing Tools:**
1. ✅ `detect-synthetic-identity` - Finds customers sharing PII (addresses Section 1.3, 2.5)
2. ✅ `get-sar-report-guidance` - Returns SAR filing guidance (FinCEN focused)
3. ✅ `get-schema` - Discovers database structure
4. ✅ `read-cypher` - Executes custom read queries
5. ✅ `write-cypher` - Executes custom write queries

### Gap Analysis
**Missing Tools for Complete SAR Creation:**

| SAR Section | Required Data | Missing Tool |
|-------------|---------------|--------------|
| 1.3 | Subject identity details | `get-customer-profile` |
| 1.6 | Previous SAR references | `find-related-sars` |
| 2.1 | Transaction history | `get-transaction-history` |
| 2.2 | Property/quantum details | `calculate-suspicious-amounts` |
| 2.3 | 5 W's analysis | `analyze-suspicious-activity` |
| 2.5 | Other parties involved | `find-related-parties` |
| 3.2 | Criminal activity patterns | `detect-criminal-patterns` |
| 4.1 | Related investigations | `find-related-cases` |
| 5.1-5.5 | Defence information | `generate-defence-statement` |
| All | Complete SAR generation | `generate-sar-report` |

---

## Tool Specifications

### Category 1: Customer & Identity Tools

#### 1. `get-customer-profile` ✨ NEW
**Purpose:** Gather complete customer information for SAR Section 1.3

**Maps to SAR Sections:** 1.3, 1.4, 2.5

**Input:**
```json
{
  "customerId": "string (required)",
  "includeRelationships": "boolean (optional, default: false)"
}
```

**Output:**
```json
{
  "customerId": "string",
  "personalDetails": {
    "firstName": "string",
    "lastName": "string",
    "dateOfBirth": "date",
    "nationality": "string"
  },
  "contactInformation": {
    "emails": ["string"],
    "phones": ["string"],
    "addresses": [
      {
        "street": "string",
        "city": "string",
        "state": "string",
        "zip": "string",
        "country": "string",
        "type": "current|previous|business|residential",
        "validFrom": "date",
        "validTo": "date"
      }
    ]
  },
  "identificationDocuments": {
    "ssn": "string",
    "driverLicense": {
      "number": "string",
      "state": "string",
      "expiryDate": "date"
    },
    "passport": {
      "number": "string",
      "country": "string",
      "expiryDate": "date"
    }
  },
  "accountInformation": {
    "accounts": [
      {
        "accountNumber": "string",
        "accountType": "string",
        "openedDate": "date",
        "status": "active|closed|frozen"
      }
    ]
  },
  "employmentDetails": {
    "occupation": "string",
    "employer": "string",
    "businessType": "string"
  },
  "relationships": {
    "beneficialOwners": ["customerId"],
    "authorizedUsers": ["customerId"],
    "linkedEntities": ["entityId"]
  }
}
```

**Neo4j Cypher:**
```cypher
MATCH (c:Customer {customerId: $customerId})
OPTIONAL MATCH (c)-[:HAS_EMAIL]->(e:Email)
OPTIONAL MATCH (c)-[:HAS_PHONE]->(p:Phone)
OPTIONAL MATCH (c)-[:HAS_SSN]->(s:SSN)
OPTIONAL MATCH (c)-[:HAS_ADDRESS]->(a:Address)
OPTIONAL MATCH (c)-[:HAS_DRIVER_LICENSE]->(dl:DriverLicense)
OPTIONAL MATCH (c)-[:HAS_PASSPORT]->(pp:Passport)
OPTIONAL MATCH (c)-[:OWNS]->(acc:Account)
OPTIONAL MATCH (c)-[:HAS_EMPLOYMENT]->(emp:Employment)
OPTIONAL MATCH (c)-[:BENEFICIAL_OWNER_OF]->(entity:Entity)
OPTIONAL MATCH (c)-[:AUTHORIZED_USER_OF]->(authAcc:Account)
RETURN c,
       collect(DISTINCT e) as emails,
       collect(DISTINCT p) as phones,
       collect(DISTINCT s) as ssns,
       collect(DISTINCT a) as addresses,
       collect(DISTINCT dl) as driverLicenses,
       collect(DISTINCT pp) as passports,
       collect(DISTINCT acc) as accounts,
       collect(DISTINCT emp) as employment,
       collect(DISTINCT entity) as entities
```

**SAR Mapping:**
- Section 1.3: Client/Recipient identification (all fields)
- Section 1.4: Activities/Location (occupation, business details)
- Section 2.5: Other parties (relationships)

---

#### 2. `find-related-parties` ✨ NEW
**Purpose:** Identify all parties involved in suspicious activity

**Maps to SAR Sections:** 2.5, 4.2

**Input:**
```json
{
  "customerId": "string (required)",
  "relationshipTypes": ["string"] (optional, default: all),
  "includeTransactionCounterparties": "boolean (optional, default: true)",
  "maxDepth": "integer (optional, default: 2)"
}
```

**Output:**
```json
{
  "relatedParties": [
    {
      "customerId": "string",
      "name": "string",
      "relationshipType": "beneficiary|authorized_user|transaction_counterparty|co_signer",
      "relationshipPath": "string",
      "dateEstablished": "date",
      "transactionCount": "integer",
      "totalAmount": "float",
      "role": "victim|suspect|unknown"
    }
  ]
}
```

**Neo4j Cypher:**
```cypher
MATCH (subject:Customer {customerId: $customerId})
MATCH path = (subject)-[*1..$maxDepth]-(related:Customer)
WHERE subject <> related
WITH related,
     path,
     relationships(path) as rels
OPTIONAL MATCH (subject)-[:OWNS]->(sAcc:Account)-[t:TRANSACTION]->(rAcc:Account)<-[:OWNS]-(related)
RETURN DISTINCT
  related.customerId as customerId,
  related.firstName + ' ' + related.lastName as name,
  [r in rels | type(r)] as relationshipPath,
  COUNT(DISTINCT t) as transactionCount,
  SUM(t.amount) as totalAmount
ORDER BY transactionCount DESC, totalAmount DESC
LIMIT 50
```

**SAR Mapping:**
- Section 2.5: Other parties involved
- Section 4.2: Relevant individuals in proceedings

---

### Category 2: Transaction & Activity Tools

#### 3. `get-transaction-history` ✨ NEW
**Purpose:** Retrieve comprehensive transaction history for SAR Section 2.1, 2.2

**Maps to SAR Sections:** 2.1, 2.2, 2.3

**Input:**
```json
{
  "customerId": "string (optional)",
  "accountNumber": "string (optional)",
  "startDate": "date (required)",
  "endDate": "date (required)",
  "minAmount": "float (optional)",
  "maxAmount": "float (optional)",
  "transactionTypes": ["string"] (optional),
  "includeCounterparties": "boolean (optional, default: true)",
  "sortBy": "date|amount|suspicionScore (optional, default: date)"
}
```

**Output:**
```json
{
  "summary": {
    "totalTransactions": "integer",
    "totalAmount": "float",
    "dateRange": {"start": "date", "end": "date"},
    "accountsInvolved": "integer"
  },
  "transactions": [
    {
      "transactionId": "string",
      "timestamp": "datetime",
      "amount": "float",
      "currency": "string",
      "type": "wire|ach|check|card|cash",
      "description": "string",
      "fromAccount": {
        "accountNumber": "string",
        "customerId": "string",
        "customerName": "string"
      },
      "toAccount": {
        "accountNumber": "string",
        "customerId": "string",
        "customerName": "string"
      },
      "location": {
        "city": "string",
        "state": "string",
        "country": "string",
        "ipAddress": "string"
      },
      "flags": ["structuring", "velocity", "round_amount", "high_risk_jurisdiction"]
    }
  ]
}
```

**Neo4j Cypher:**
```cypher
MATCH (c:Customer {customerId: $customerId})-[:OWNS]->(a:Account)
MATCH (a)-[t:TRANSACTION]->(targetAcc:Account)
WHERE t.timestamp >= datetime($startDate)
  AND t.timestamp <= datetime($endDate)
  AND ($minAmount IS NULL OR t.amount >= $minAmount)
  AND ($maxAmount IS NULL OR t.amount <= $maxAmount)
OPTIONAL MATCH (targetAcc)<-[:OWNS]-(targetCustomer:Customer)
RETURN
  t.transactionId as transactionId,
  t.timestamp as timestamp,
  t.amount as amount,
  t.currency as currency,
  t.type as type,
  t.description as description,
  a.accountNumber as fromAccount,
  c.customerId as fromCustomerId,
  c.firstName + ' ' + c.lastName as fromCustomerName,
  targetAcc.accountNumber as toAccount,
  targetCustomer.customerId as toCustomerId,
  targetCustomer.firstName + ' ' + targetCustomer.lastName as toCustomerName,
  t.location as location,
  t.flags as flags
ORDER BY t.timestamp DESC
```

**SAR Mapping:**
- Section 2.1: What has been heard or observed
- Section 2.2: Nature of criminal property and quantum (chronological sequence)
- Section 2.3: Reason for suspicion (when/where/how)

---

#### 4. `calculate-suspicious-amounts` ✨ NEW
**Purpose:** Quantify proceeds of crime for SAR Section 2.2

**Maps to SAR Sections:** 2.2

**Input:**
```json
{
  "customerId": "string (optional)",
  "accountNumber": "string (optional)",
  "transactionIds": ["string"] (optional),
  "startDate": "date (required)",
  "endDate": "date (required)",
  "calculationMethod": "sum|net|peak|cumulative (optional, default: sum)"
}
```

**Output:**
```json
{
  "quantum": {
    "totalSuspiciousAmount": "float",
    "currency": "string",
    "calculationMethod": "string",
    "isEstimate": "boolean",
    "confidence": "high|medium|low"
  },
  "breakdown": {
    "inflows": "float",
    "outflows": "float",
    "netAmount": "float",
    "peakBalance": "float"
  },
  "propertyDetails": [
    {
      "propertyType": "cash|property|cryptocurrency|other",
      "amount": "float",
      "location": "string",
      "description": "string"
    }
  ],
  "narrative": "string (auto-generated description)"
}
```

**SAR Mapping:**
- Section 2.2: Nature of criminal property and quantum
- Section 5.4: Criminal property description (for defence)

---

#### 5. `analyze-suspicious-activity` ✨ NEW
**Purpose:** Answer the 5 W's for SAR Section 2.3

**Maps to SAR Sections:** 2.3

**Input:**
```json
{
  "customerId": "string (required)",
  "startDate": "date (required)",
  "endDate": "date (required)",
  "activityType": "transaction|identity|network|all (optional, default: all)"
}
```

**Output:**
```json
{
  "fiveWs": {
    "who": {
      "primarySubject": {
        "customerId": "string",
        "name": "string",
        "role": "main_subject"
      },
      "associatedSubjects": [
        {
          "customerId": "string",
          "name": "string",
          "role": "beneficiary|accomplice|victim|unknown"
        }
      ]
    },
    "what": {
      "activityType": "string",
      "description": "string",
      "patterns": ["structuring", "layering", "smurfing"]
    },
    "when": {
      "firstDetected": "date",
      "lastOccurrence": "date",
      "frequency": "daily|weekly|monthly",
      "timeline": [
        {
          "date": "date",
          "event": "string"
        }
      ]
    },
    "where": {
      "locations": ["string"],
      "accounts": ["string"],
      "geographicAnomalies": ["string"]
    },
    "why": {
      "suspicionReasons": ["string"],
      "redFlags": ["string"],
      "riskScore": "float",
      "confidence": "high|medium|low"
    }
  },
  "narrative": "string (auto-generated)"
}
```

**SAR Mapping:**
- Section 2.3: Reason for suspicion (complete 5 W's)

---

### Category 3: Investigation & Case Management Tools

#### 6. `find-related-sars` ✨ NEW
**Purpose:** Find previous SAR filings for the same subject

**Maps to SAR Sections:** 1.6, 1.7

**Input:**
```json
{
  "customerId": "string (optional)",
  "accountNumber": "string (optional)",
  "dateRange": {
    "start": "date (optional)",
    "end": "date (optional)"
  },
  "includeRelatedCustomers": "boolean (optional, default: false)"
}
```

**Output:**
```json
{
  "previousSARs": [
    {
      "sarId": "string",
      "ukURN": "string",
      "filingDate": "date",
      "customerId": "string",
      "accountNumber": "string",
      "activityType": "string",
      "status": "filed|under_investigation|closed",
      "relatedCustomers": ["string"]
    }
  ],
  "suggestedUpdateReason": "string (if this is an update)"
}
```

**Neo4j Cypher:**
```cypher
MATCH (c:Customer {customerId: $customerId})
OPTIONAL MATCH (c)-[:SUBJECT_OF]->(sar:SAR)
WHERE ($startDate IS NULL OR sar.filingDate >= datetime($startDate))
  AND ($endDate IS NULL OR sar.filingDate <= datetime($endDate))
OPTIONAL MATCH (sar)<-[:SUBJECT_OF]-(relatedCustomer:Customer)
WHERE relatedCustomer <> c
RETURN
  sar.sarId as sarId,
  sar.ukURN as ukURN,
  sar.filingDate as filingDate,
  sar.activityType as activityType,
  sar.status as status,
  collect(DISTINCT relatedCustomer.customerId) as relatedCustomers
ORDER BY sar.filingDate DESC
```

**SAR Mapping:**
- Section 1.6: Previous report reference (date and URN)
- Section 1.7: Reason for additional update

---

#### 7. `find-related-cases` ✨ NEW
**Purpose:** Identify related fraud investigations and alerts

**Maps to SAR Sections:** 4.1, 4.2, 4.3

**Input:**
```json
{
  "customerId": "string (required)",
  "includeAlerts": "boolean (optional, default: true)",
  "includeCases": "boolean (optional, default: true)",
  "status": "open|closed|all (optional, default: all)"
}
```

**Output:**
```json
{
  "relatedAlerts": [
    {
      "alertId": "string",
      "type": "synthetic_identity|account_takeover|money_laundering",
      "createdAt": "date",
      "status": "new|investigating|closed",
      "severity": "low|medium|high|critical"
    }
  ],
  "relatedCases": [
    {
      "caseId": "string",
      "type": "fraud|aml|kyc",
      "createdAt": "date",
      "status": "open|closed",
      "investigator": "string",
      "lawEnforcementAgency": "string",
      "referenceNumber": "string"
    }
  ],
  "otherProceedings": [
    {
      "proceedingType": "civil|criminal|administrative",
      "agency": "Metropolitan Police|HMRC|SFO|FCA|Interpol",
      "referenceNumber": "string",
      "status": "ongoing|completed",
      "outcome": "string"
    }
  ]
}
```

**Neo4j Cypher:**
```cypher
MATCH (c:Customer {customerId: $customerId})
OPTIONAL MATCH (c)-[:SUBJECT_OF]->(alert:Alert)
WHERE $includeAlerts = true
  AND ($status = 'all' OR alert.status = $status)
OPTIONAL MATCH (c)-[:SUBJECT_OF]->(case:Case)
WHERE $includeCases = true
  AND ($status = 'all' OR case.status = $status)
OPTIONAL MATCH (c)-[:INVOLVED_IN]->(proceeding:Proceeding)
RETURN
  collect(DISTINCT alert) as alerts,
  collect(DISTINCT case) as cases,
  collect(DISTINCT proceeding) as proceedings
```

**SAR Mapping:**
- Section 4.1: Reports to other authorities
- Section 4.2: Relevant individuals in proceedings
- Section 4.3: What has happened (dismissal, investigation status)

---

### Category 4: Pattern Detection Tools

#### 8. `detect-criminal-patterns` ✨ NEW
**Purpose:** Identify specific criminal activity patterns

**Maps to SAR Sections:** 3.2, 3.3, 3.4

**Input:**
```json
{
  "customerId": "string (required)",
  "patternTypes": ["money_laundering", "terrorist_financing", "fraud", "all"],
  "startDate": "date (required)",
  "endDate": "date (required)",
  "sensitivity": "low|medium|high (optional, default: medium)"
}
```

**Output:**
```json
{
  "detectedPatterns": [
    {
      "patternType": "string",
      "criminalActivity": "fraud_by_misrepresentation|theft|bribery|tax_evasion",
      "legislation": "Fraud Act|Theft Act|Bribery Act|Tax Acts",
      "mlTfOffence": "concealing|acquiring|use|possession|arrangement",
      "sarCode": "XXD5XX|XXD6XX",
      "description": "string",
      "confidence": "float",
      "evidence": [
        {
          "type": "transaction|identity|network",
          "description": "string",
          "timestamp": "date"
        }
      ]
    }
  ],
  "riskAssessment": {
    "overallRisk": "low|medium|high|critical",
    "pocaSection": "327|328|329 (if applicable)",
    "ta2000Section": "string (if applicable)"
  }
}
```

**Pattern Detection Rules:**
- **Structuring**: Multiple transactions just below reporting thresholds
- **Layering**: Complex chains of transactions to obscure source
- **Smurfing**: Multiple small deposits from different sources
- **Rapid movement**: Funds moving through accounts quickly
- **Circular flows**: Money returning to source
- **Shell company activity**: Transactions with no clear business purpose

**SAR Mapping:**
- Section 3.2: Criminal activity description
- Section 3.3: Money laundering / terrorist financing offence
- Section 3.4: SAR code for ML/TF activity

---

### Category 5: SAR Generation & Defence Tools

#### 9. `generate-defence-statement` ✨ NEW
**Purpose:** Generate defence against money laundering statement

**Maps to SAR Sections:** 5.1, 5.2, 5.3, 5.4, 5.5

**Input:**
```json
{
  "customerId": "string (required)",
  "prohibitedAct": {
    "pocaSection": "327|328|329 (required)",
    "description": "string (required)",
    "transactionDetails": {
      "vendor": "string",
      "purchaser": "string",
      "considerationAmount": "float",
      "transactionValue": "float"
    }
  },
  "timeline": [
    {
      "date": "date",
      "action": "string"
    }
  ]
}
```

**Output:**
```json
{
  "defenceStatement": {
    "sarCode": "XXS99XX",
    "pocaSection": "string",
    "prohibitedActDescription": "string",
    "relevantIndividuals": [
      {
        "customerId": "string",
        "name": "string",
        "role": "string"
      }
    ],
    "rationale": "string",
    "criminalPropertyDescription": "string",
    "propertyWhereabouts": "string",
    "propertyValue": {
      "amount": "float",
      "isEstimate": "boolean"
    },
    "nextSteps": [
      {
        "action": "string",
        "targetDate": "date"
      }
    ]
  },
  "formattedStatement": "string (ready for SAR Section 5)"
}
```

**SAR Mapping:**
- Section 5.1: Prohibited act
- Section 5.2: Relevant individuals
- Section 5.3: Rationale
- Section 5.4: Criminal property description
- Section 5.5: Next steps

---

#### 10. `generate-sar-report` ✨ NEW - **PRIMARY TOOL**
**Purpose:** Orchestrate all other tools to create a complete SAR

**Maps to SAR Sections:** ALL (1.1 - 6.2)

**Input:**
```json
{
  "customerId": "string (required)",
  "reportType": "money_laundering|terrorist_financing (required)",
  "professionalService": "string (required)",
  "dateRange": {
    "start": "date (required)",
    "end": "date (required)"
  },
  "includeDefence": "boolean (optional, default: false)",
  "defenceDetails": "object (required if includeDefence=true)",
  "mlroDetails": {
    "name": "string (required)",
    "phone": "string (required)",
    "email": "string (required)"
  }
}
```

**Output:**
```json
{
  "sar": {
    "metadata": {
      "generatedAt": "datetime",
      "reportType": "money_laundering|terrorist_financing",
      "customerId": "string"
    },
    "section1_introduction": {
      "1.1": "string (introduction summary)",
      "1.2": "string (professional services)",
      "1.3": "object (client details)",
      "1.4": "string (activities/location)",
      "1.5": "string (situation)",
      "1.6": "string (previous report reference)",
      "1.7": "string (update reason)"
    },
    "section2_suspicion": {
      "2.1": "string (what observed)",
      "2.2": "string (criminal property quantum)",
      "2.3": "object (5 W's analysis)",
      "2.4": "string (glossary code)",
      "2.5": "array (other parties)",
      "2.6": "string (other information)"
    },
    "section3_disclosure": {
      "3.1": "string (link to criminal activity)",
      "3.2": "string (criminal activity description)",
      "3.3": "string (ML/TF offence)",
      "3.4": "string (SAR code)"
    },
    "section4_other": {
      "4.1": "object (other authorities)",
      "4.2": "array (relevant individuals)",
      "4.3": "string (what happened)"
    },
    "section5_defence": {
      "5.1": "string (prohibited act)",
      "5.2": "array (relevant individuals)",
      "5.3": "string (rationale)",
      "5.4": "string (criminal property)",
      "5.5": "array (next steps)"
    },
    "section6_contact": {
      "6.1": "string (MLRO name)",
      "6.2": "object (contact details)"
    }
  },
  "supportingEvidence": {
    "transactionRecords": "array",
    "identityDocuments": "array",
    "relatedInvestigations": "array"
  },
  "formattedReport": "string (complete markdown SAR)"
}
```

**Tool Orchestration:**
1. Call `get-customer-profile` for Section 1.3, 1.4
2. Call `find-related-sars` for Section 1.6, 1.7
3. Call `get-transaction-history` for Section 2.1, 2.2
4. Call `calculate-suspicious-amounts` for Section 2.2
5. Call `analyze-suspicious-activity` for Section 2.3
6. Call `find-related-parties` for Section 2.5
7. Call `detect-criminal-patterns` for Section 3.2, 3.3, 3.4
8. Call `find-related-cases` for Section 4.1, 4.2, 4.3
9. Call `generate-defence-statement` for Section 5 (if requested)
10. Format all data into complete SAR report

**SAR Mapping:**
- ALL SECTIONS: Complete orchestration

---

## Implementation Priority

### Phase 1: Core Evidence Gathering (Weeks 1-2)
1. ✨ `get-customer-profile` - Most critical, needed for every SAR
2. ✨ `get-transaction-history` - Core evidence for suspicious activity
3. ✨ `analyze-suspicious-activity` - Provides the "why" for suspicion

### Phase 2: Context & Relationships (Weeks 3-4)
4. ✨ `find-related-parties` - Identifies all involved parties
5. ✨ `calculate-suspicious-amounts` - Quantifies proceeds of crime
6. ✨ `find-related-sars` - Historical context

### Phase 3: Investigation Integration (Weeks 5-6)
7. ✨ `find-related-cases` - Links to existing investigations
8. ✨ `detect-criminal-patterns` - Identifies criminal activity types

### Phase 4: Report Generation (Weeks 7-8)
9. ✨ `generate-defence-statement` - Defence section (if needed)
10. ✨ `generate-sar-report` - Complete orchestration tool

---

## Neo4j Data Model Requirements

### New Node Types Needed

#### SAR Node
```cypher
CREATE (sar:SAR {
  sarId: string,
  ukURN: string,           // UK Unique Reference Number
  filingDate: datetime,
  reportType: string,      // "money_laundering" | "terrorist_financing"
  status: string,          // "draft" | "filed" | "under_investigation" | "closed"
  activityType: string,
  sarCode: string,         // "XXD5XX", "XXD6XX", etc.
  glossaryCode: string,
  pocaSection: string,     // "327" | "328" | "329"
  includesDefence: boolean,
  mlroName: string,
  createdAt: datetime,
  updatedAt: datetime
})
```

#### Case Node (Investigation)
```cypher
CREATE (case:Case {
  caseId: string,
  type: string,           // "fraud" | "aml" | "kyc"
  status: string,         // "open" | "closed"
  createdAt: datetime,
  investigator: string,
  lawEnforcementAgency: string,
  referenceNumber: string,
  outcome: string
})
```

#### Proceeding Node (Legal Actions)
```cypher
CREATE (proc:Proceeding {
  proceedingId: string,
  type: string,          // "civil" | "criminal" | "administrative"
  agency: string,        // "Metropolitan Police" | "HMRC" | "SFO" | etc.
  referenceNumber: string,
  status: string,        // "ongoing" | "completed"
  outcome: string,
  startDate: datetime,
  endDate: datetime
})
```

### New Relationships Needed

```cypher
// SAR relationships
(Customer)-[:SUBJECT_OF]->(SAR)
(SAR)-[:REFERENCES_SAR]->(SAR)  // Links to previous SARs
(SAR)-[:INVOLVES_ACCOUNT]->(Account)
(SAR)-[:INVOLVES_TRANSACTION]->(Transaction)

// Investigation relationships
(Customer)-[:SUBJECT_OF]->(Alert)
(Customer)-[:SUBJECT_OF]->(Case)
(Customer)-[:INVOLVED_IN]->(Proceeding)
(Alert)-[:ESCALATED_TO]->(Case)
(Case)-[:GENERATED_SAR]->(SAR)
```

---

## Testing Strategy

### Unit Tests
Each tool must have:
- Input validation tests
- Cypher query correctness tests
- Output format tests
- Edge case handling

### Integration Tests
- Test tool orchestration in `generate-sar-report`
- Test with complete customer profiles
- Test with missing data scenarios

### End-to-End Tests
- Generate complete SAR from synthetic fraud scenario
- Validate against SAR requirements checklist
- Compare with template examples

---

## Documentation Requirements

### Tool Documentation
Each tool needs:
1. **Description** - When to use, what it detects
2. **Input Schema** - All parameters with examples
3. **Output Schema** - Expected return structure
4. **SAR Mapping** - Which sections it addresses
5. **Example Usage** - Claude Desktop prompts
6. **Cypher Queries** - Full query documentation

### User Guides
1. **SAR Creation Workflow** - Step-by-step guide
2. **Prompt Examples** - Natural language queries
3. **Troubleshooting** - Common issues and solutions

---

## Success Metrics

### Functional Metrics
- ✅ All SAR sections can be populated from tool outputs
- ✅ `generate-sar-report` creates compliance-ready SARs
- ✅ Tools work with realistic fraud detection data

### Performance Metrics
- Transaction history query < 2 seconds (10K transactions)
- Customer profile query < 500ms
- Full SAR generation < 10 seconds

### Quality Metrics
- Generated SARs pass compliance review
- All required fields populated
- Narratives are clear and factual
- Evidence is properly documented

---

## Open Questions

1. **SAR Storage**: Should we store generated SARs in Neo4j or export only?
2. **Update Workflow**: How to handle SAR amendments/updates?
3. **Multi-Customer SARs**: How to handle fraud rings with multiple subjects?
4. **Evidence Attachments**: How to reference supporting documents?
5. **Glossary Codes**: Do we need a complete glossary code database?
6. **Regulatory Codes**: Should we validate SAR codes against official lists?

---

## Next Steps

1. **Review with stakeholders** - Validate tool specifications
2. **Prioritize MVP tools** - Select Phase 1 tools
3. **Create implementation tickets** - One per tool
4. **Set up test data** - Synthetic fraud scenarios
5. **Begin implementation** - Start with `get-customer-profile`

---

*This specification maps directly to [SAR-Requirements.md](SAR-Requirements.md) to ensure complete coverage of all SAR sections.*
