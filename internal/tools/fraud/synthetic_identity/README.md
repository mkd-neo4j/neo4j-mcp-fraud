# Synthetic Identity Fraud Detection Tool

## Overview

The `detect-synthetic-identity` tool identifies potential synthetic identity fraud by finding customers who share multiple identity attributes (email addresses, phone numbers, passport information) with a target customer.

## What is Synthetic Identity Fraud?

Synthetic identity fraud occurs when fraudsters combine real and fake information to create new identities. Unlike traditional identity theft where a single victim's identity is stolen, synthetic identities are fabricated using:
- Real Social Security Numbers (often from children or deceased individuals)
- Fake names, addresses, and contact information
- Mix of legitimate and fraudulent documents

In graph terms, synthetic identity fraud manifests as **multiple customer accounts sharing the same identity attributes** (phones, emails, passports).

## What This Tool Detects

This tool finds customers who share identity attributes with the target customer by examining:

1. **Shared Email Addresses**: Multiple customers using the same email
2. **Shared Phone Numbers**: Multiple customers using the same phone
3. **Shared Passports**: Multiple customers claiming the same passport information

## Data Model

Based on the [Neo4j Transaction & Account Base Model](../../DATA_MODEL.md), this tool queries:

### Nodes
- `:Customer` - Customer accounts in the system
- `:Email` - Email addresses
- `:Phone` - Phone numbers
- `:Passport` - Passport documents

### Relationships
- `(:Customer)-[:HAS_EMAIL]->(:Email)`
- `(:Customer)-[:HAS_PHONE]->(:Phone)`
- `(:Customer)-[:HAS_PASSPORT]->(:Passport)`

## Cypher Query Pattern

```cypher
MATCH (target:Customer {customerId: $customerId})
MATCH (target)-[r:HAS_EMAIL|HAS_PHONE|HAS_PASSPORT]->(identifier)
MATCH (identifier)<-[r2:HAS_EMAIL|HAS_PHONE|HAS_PASSPORT]-(other:Customer)
WHERE target.customerId <> other.customerId
WITH other, collect(DISTINCT {...}) as sharedAttributes
WHERE size(sharedAttributes) >= $minSharedAttributes
RETURN ...
```

## Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `customerId` | string | Yes | - | Customer ID to investigate |
| `minSharedAttributes` | integer | No | 2 | Minimum number of shared attributes to flag |

## Return Format

Returns a JSON array of customers sharing identity attributes:

```json
[
  {
    "customerId": "CUS456",
    "firstName": "Jane",
    "lastName": "Doe",
    "sharedAttributes": [
      {"type": "HAS_EMAIL", "identifier": "jane@example.com"},
      {"type": "HAS_PHONE", "identifier": "+44123456789"}
    ],
    "sharedAttributeCount": 2
  }
]
```

## Risk Indicators

| Shared Attributes | Risk Level | Interpretation |
|-------------------|------------|----------------|
| 3+ | CRITICAL | Strong indicator of fraud ring or synthetic identity operation |
| 2 | HIGH | Likely synthetic identity pattern, warrants investigation |
| 1 | MEDIUM | May be legitimate (family members, business contacts) |

## Investigation Workflow

1. **Run the tool** on a suspect customer ID
2. **Review results**: Examine which specific attributes are shared
3. **Assess pattern**: Are identities shared across unrelated accounts?
4. **Check transactions**: Investigate transaction patterns of linked customers
5. **Follow up**: Use additional fraud tools on connected customers

## Example Use Cases

### Use Case 1: New Customer Onboarding
**Scenario**: A new customer application seems suspicious
**Action**: Run `detect-synthetic-identity` with the applicant's customer ID
**Investigation**: If 2+ other customers share their email/phone, flag for manual review

### Use Case 2: Account Takeover Investigation
**Scenario**: Investigating a potential account takeover
**Action**: Check if the account shares identity attributes with known fraudsters
**Investigation**: Shared identifiers may indicate the same fraud ring

### Use Case 3: Transaction Monitoring Alert
**Scenario**: High-value transaction triggered an alert
**Action**: Run synthetic identity check on both sender and receiver
**Investigation**: If either party has synthetic identity patterns, escalate case

## Integration with Other Tools

This tool works well in combination with:
- **Device sharing detection** - Find customers sharing the same devices
- **Circular flow detection** - Identify money laundering patterns
- **Bad actor proximity** - Check connections to known fraudsters

## Performance Considerations

- Query is optimized with indexes on:
  - `Customer.customerId` (NODE KEY constraint)
  - `Email.address` (NODE KEY constraint)
  - `Phone.number` (NODE KEY constraint)
  - `Passport.passportNumber` + `issuingCountry` (composite NODE KEY)
- Results limited to 100 customers to prevent performance issues
- Uses read-only query execution

## References

- [Neo4j Synthetic Identity Fraud Use Case](https://neo4j.com/developer/industry-use-cases/finserv/retail-banking/synthetic-identity-fraud/)
- [Fraud Detection Data Model](../../DATA_MODEL.md)
- [Transaction & Account Base Model](https://neo4j.com/developer/industry-use-cases/_attachments/transaction-base-model.txt)
