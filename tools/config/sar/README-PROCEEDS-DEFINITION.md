# Proceeds of Crime Definition Tool

## Overview

The `define-proceeds-of-crime` tool provides a **user-controlled, 7-phase conversational workflow** for defining "proceeds of crime" in SAR investigations.

**Location:** [tools/config/sar/define-proceeds-of-crime.yaml](define-proceeds-of-crime.yaml)

---

## Why This Tool Exists

### The Critical Problem

**Legal Definition (POCA 2002):**
> "Proceeds of crime include any monies or assets deriving **directly or indirectly** from criminal conduct"

Getting this definition wrong can:
- ❌ Invalidate the entire SAR investigation
- ❌ Result in rejected SAR filings
- ❌ Lead to regulatory sanctions
- ❌ Compromise criminal prosecutions

### The Risk with Automated Tools

Traditional fraud detection tools that automatically calculate "suspicious amounts" are **dangerous** because:
- They make assumptions about which transactions are proceeds
- They don't require user justification for inclusions/exclusions
- They lack audit trails of decision-making
- They can't distinguish between criminal proceeds and legitimate transactions

**Example of what can go wrong:**
```
❌ Tool automatically calculates: "Total suspicious transactions: $500,000"

But this includes:
- $300,000 in legitimate business expenses
- $150,000 to creditors for documented debts
- Only $50,000 are actual criminal proceeds

Result: SAR is rejected, investigation is compromised
```

---

## How This Tool Solves It

### Core Principle: **100% User Control**

The LLM acts as an **assistant**, not a **decision-maker**:

| Traditional Tools | This Tool |
|------------------|-----------|
| ❌ Automatically classifies transactions | ✅ User explicitly classifies each transaction |
| ❌ Makes assumptions about proceeds | ✅ User provides evidence-based reasoning |
| ❌ No audit trail of decisions | ✅ Complete audit trail with timestamps |
| ❌ "Black box" calculations | ✅ Transparent, user-documented process |
| ❌ No confirmation gates | ✅ Multiple confirmation gates at each phase |

---

## The 7-Phase Workflow

### Phase 1: Investigation Scope Definition
**User Controls:** What criminal activity, who is involved, what time period

```
User provides:
- Criminal activity type (fraud, theft, tax evasion, corruption)
- Suspect customer/account IDs
- Investigation date range
- Known criminal transaction IDs (if any)

LLM confirms back: "This is your investigation scope. Is this correct?"
User must confirm: "Yes" before proceeding
```

**Key Safeguard:** User defines scope, LLM cannot proceed without confirmation

---

### Phase 2: Transaction Retrieval
**LLM Task:** Retrieve ALL transactions in scope (no filtering)

```
LLM executes:
1. Calls get-schema to understand database
2. Calls get-transaction-history with user's scope
3. Retrieves complete transaction dataset

LLM presents:
- Full transaction list (47 transactions found)
- Summary statistics ($458,923 total)
- Pattern highlights (informational only):
  - 3 transactions flagged for structuring
  - 4 to high-risk jurisdictions
  - 8 round amounts

CRITICAL: LLM highlights patterns but makes NO classifications
```

**Key Safeguard:** User sees ALL data before making decisions

---

### Phase 3: Direct Proceeds Classification
**User Controls:** Which transactions are DIRECT proceeds (1-hop from criminal act)

```
User provides:
{
  "directProceedsTransactionIds": ["tx-001", "tx-002", "tx-005"],
  "reasoning": "These are fraudulent wire transfers directly from victim
                accounts to suspect's account on [dates]. Evidence:
                victim statements, bank records, emails proving fraud.",
  "evidenceSources": [
    "Victim witness statement dated 2024-02-15",
    "Bank records showing transfers",
    "Email correspondence proving fraudulent misrepresentation"
  ]
}

LLM confirms back:
"You have classified 3 transactions as DIRECT proceeds:
 - tx-001: $50,000 (2024-01-15)
 - tx-002: $30,000 (2024-01-20)
 - tx-005: $20,000 (2024-02-01)
 Total: $100,000

 Your reasoning: [user's exact words]

 Is this correct? (yes/no/modify)"

User must confirm: "Yes" before proceeding
```

**Key Safeguards:**
- User must provide reasoning for EACH transaction
- User must specify evidence sources
- LLM confirms back user's exact words
- User must explicitly approve

---

### Phase 4: Indirect Proceeds Path Discovery
**User Controls:** Whether to trace downstream flows and how many hops

```
LLM asks: "Do you want to trace where the direct proceeds went?"

User options:
1. "Yes, trace all outflows (unlimited hops)"
2. "Yes, trace up to N hops" (user specifies N)
3. "Yes, but only to accounts controlled by suspect"
4. "No, only count direct proceeds"

If user selects yes:
LLM executes variable-length path query:
  MATCH path = (source:Account)-[:PERFORMS|BENEFITS_TO*1..5]->()
  WHERE source has direct proceeds
  RETURN all downstream transaction chains

LLM presents:
Chain 1: Account A → tx-010 ($48k) → Account B → tx-011 ($46k) → Account C
Chain 2: Account A → tx-012 ($10k) → Account D (ABC Company - creditor)
Chain 3: Account A → tx-015 ($42k) → Account E (Shell Co)

CRITICAL: "These are POTENTIAL indirect proceeds - you must classify them"
```

**Key Safeguard:** User decides whether to trace and how deep

---

### Phase 5: Indirect Proceeds Classification
**User Controls:** Which downstream transactions are INDIRECT proceeds

```
User provides:
{
  "indirectProceedsMethod": "specific_transactions",
  "transactionIds": ["tx-010", "tx-011", "tx-015"],
  "reasoning": "These funds from original fraud were layered through
                shell company accounts. Evidence shows suspect controlled
                these intermediaries based on device fingerprints, IP
                addresses, shared identity documents.",
  "evidenceSources": [
    "Device fingerprint analysis (same device as suspect)",
    "IP logs match suspect's home address",
    "Beneficial ownership documents show suspect control"
  ],
  "excludedTransactionIds": ["tx-012", "tx-013"],
  "exclusionReasoning": "These went to legitimate creditors for
                         documented debts. No evidence suspect
                         controlled these entities.",
  "exclusionEvidence": [
    "ABC Company is registered UK company with no links to suspect",
    "Documentary evidence of legitimate debt (loan agreement)"
  ]
}

LLM confirms back:
"INDIRECT PROCEEDS:
 - tx-010: $48,000 (Chain 1, Hop 1)
 - tx-011: $46,000 (Chain 1, Hop 2)
 - tx-015: $42,000 (Chain 3, Final destination)
 Total: $136,000

 Your reasoning: [user's exact words]

 EXCLUDED:
 - tx-012: $10,000 (Legitimate creditor payment)
 - tx-013: $5,000 (Legitimate creditor payment)

 Exclusion reasoning: [user's exact words]

 SUMMARY:
 - Direct: 3 transactions = $100,000
 - Indirect: 3 transactions = $136,000
 - Excluded: 2 transactions = $15,000

 Is this correct? (yes/no/modify)"

User must confirm: "Yes" before proceeding
```

**Key Safeguards:**
- User must classify each downstream transaction
- User must provide evidence for inclusions
- User must document WHY transactions are excluded
- Exclusion reasoning is as important as inclusion reasoning

---

### Phase 6: Quantum Calculation Method
**User Controls:** How to calculate the final amount

```
LLM presents options:
1. Sum: Total of all transactions ($100k + $136k = $236k)
2. Net: Inflows minus outflows
3. Peak: Maximum balance reached
4. Final Destination: Amount at end of chain ($45k after fees)
5. Custom: User defines

User selects:
{
  "calculationMethod": "sum",
  "reasoning": "I selected sum because we need to demonstrate the
                total volume of criminal proceeds moved through the
                banking system, not just what was retained. This shows
                the full scale of the money laundering operation.",
  "quantum": {
    "directProceeds": 100000.00,
    "indirectProceeds": 136000.00,
    "totalProceeds": 236000.00,
    "isEstimate": false,
    "confidence": "high"
  }
}

LLM confirms back:
"Method: Sum (total volume)
 Your reasoning: [user's exact words]

 Calculation:
 - Direct: $100,000
 - Indirect: $136,000
 - Total: $236,000

 Is Estimate: No
 Confidence: High

 Is this correct? (yes/no/change method)"

User must confirm: "Yes" before proceeding
```

**Key Safeguard:** User justifies calculation method selection

---

### Phase 7: Final Confirmation & Documentation
**User Controls:** Final review and sign-off

```
LLM presents:
# PROCEEDS OF CRIME DEFINITION - FINAL SUMMARY

[Complete summary with all phases, decisions, reasoning, evidence]

## Audit Trail
- Phase 1: 2024-03-15 09:30:00 - Scope defined
- Phase 2: 2024-03-15 09:45:00 - 47 transactions retrieved
- Phase 3: 2024-03-15 10:15:00 - 3 direct proceeds classified
- Phase 4: 2024-03-15 10:30:00 - 3 chains discovered
- Phase 5: 2024-03-15 10:45:00 - 3 indirect classified, 2 excluded
- Phase 6: 2024-03-15 10:55:00 - Sum method selected
- Phase 7: 2024-03-15 11:00:00 - Final confirmation

LLM asks:
"Is this your final definition of 'proceeds of crime' for this investigation?

By confirming, you certify that:
- All transaction classifications are based on evidence
- Reasoning for inclusions and exclusions is documented
- Quantum calculation method is appropriate
- This definition is ready for SAR Section 2.2
- You accept responsibility for this proceeds definition

Type 'CONFIRM' to finalize, or 'MODIFY [phase]' to return to a phase."

User types: "CONFIRM"

Output:
{
  "investigationScope": {...},
  "directProceeds": {...},
  "indirectProceeds": {...},
  "excludedTransactions": {...},
  "quantumCalculation": {...},
  "auditTrail": [...],
  "sarSection2_2Ready": true,
  "userConfirmed": true,
  "confirmedAt": "2024-03-15T11:00:00Z",
  "digitalSignature": "hash_of_all_decisions"
}
```

**Key Safeguard:** Complete audit trail with user's explicit final confirmation

---

## Key Safeguards Summary

### 1. No Automated Decisions
✅ LLM NEVER classifies transactions without your explicit approval
✅ All pattern highlighting is informational only
✅ System presents data, user makes decisions

### 2. Reasoning Required
✅ You must document WHY for each classification
✅ Evidence sources must be specified
✅ Exclusions must be justified (why NOT proceeds)

### 3. Multiple Confirmation Gates
✅ Scope definition confirmed (Phase 1)
✅ Direct proceeds confirmed (Phase 3)
✅ Indirect proceeds confirmed (Phase 5)
✅ Calculation method confirmed (Phase 6)
✅ Final proceeds definition confirmed (Phase 7)

### 4. Complete Audit Trail
✅ Timestamp for every phase completion
✅ Capture of all user decisions verbatim
✅ Documentation of reasoning at each step
✅ Proof of user confirmation gates passed

### 5. Reversibility
✅ Can return to any previous phase
✅ Can modify classifications at any time
✅ Draft saves before final confirmation
✅ Clear "last modified" tracking

### 6. Evidence-Based
✅ Tie each classification to specific evidence
✅ Document evidence sources
✅ Explain exclusions with evidence
✅ Audit trail includes evidence references

### 7. SAR-Compliant Output
✅ Ready for SAR Section 2.2
✅ Meets chronological sequence requirement
✅ Includes quantum with confidence level
✅ Documents property location
✅ States if estimate (with reasoning)

---

## What the LLM CAN and CANNOT Do

### LLM CAN:
✅ Execute queries you request
✅ Present transaction data for your review
✅ Highlight patterns (structuring, layering) as informational
✅ Trace money flows through graph paths
✅ Confirm back your decisions
✅ Document your reasoning verbatim
✅ Generate SAR-ready narratives from your classifications

### LLM CANNOT:
❌ Classify transactions as proceeds without your approval
❌ Make assumptions about which transactions are criminal
❌ Decide calculation methods on your behalf
❌ Exclude transactions without your justification
❌ Finalize proceeds definition without your confirmation

---

## Example Usage

### User's Perspective:

**You say:** "I need to define proceeds for a fraud investigation"

**LLM guides you through:**
1. Define investigation scope (you control what to investigate)
2. Review all transactions (you see everything before deciding)
3. Classify direct proceeds (you mark which transactions, provide evidence)
4. Discover indirect paths (you decide whether to trace and how far)
5. Classify indirect proceeds (you mark which downstream transactions, explain exclusions)
6. Select calculation method (you choose how to quantify and justify)
7. Confirm final definition (you review complete audit trail and sign off)

**Result:** You have a fully documented, evidence-based proceeds definition that:
- ✅ You control 100%
- ✅ Documents your reasoning
- ✅ Has complete audit trail
- ✅ Is ready for SAR Section 2.2
- ✅ Meets regulatory requirements

---

## Integration with Other SAR Tools

After completing this workflow, the proceeds definition feeds into:

- **`get-transaction-history`**: Use your classified transaction IDs to retrieve detailed records
- **`get-customer-profile`**: Cross-reference proceeds with customer identity data
- **`find-related-parties`**: Identify other individuals involved in handling proceeds
- **`generate-sar-report`**: Use this proceeds definition for Section 2.2

---

## Regulatory Compliance

This tool meets requirements for:

- **POCA 2002** - Proceeds of Crime Act
- **SAR Section 2.2** - Nature of criminal property and quantum
- **NCA SAR Template** - UK reporting standards
- **FinCEN SAR** - US reporting standards

**Key compliance features:**
- User-controlled classification (no automated assumptions)
- Evidence-based reasoning required
- Complete audit trail of all decisions
- Explicit confirmation gates at each phase
- Handles both direct and indirect proceeds
- Documents exclusions (what is NOT proceeds)
- Multiple calculation methods supported
- Clear statement if quantum is estimate

---

## When to Use This Tool

**Use this tool when:**
- ✅ You need to define proceeds for a SAR investigation
- ✅ You need complete control over transaction classifications
- ✅ You need an audit trail of your decision-making
- ✅ You need to document reasoning for inclusions AND exclusions
- ✅ You need to handle both direct and indirect proceeds
- ✅ You need SAR-compliant output for Section 2.2

**Do NOT use automated tools that:**
- ❌ Automatically calculate "suspicious amounts" without user control
- ❌ Make assumptions about what constitutes proceeds
- ❌ Lack audit trails
- ❌ Don't require user justification for decisions

---

## Critical Success Factors

### 1. User Must Provide Evidence-Based Reasoning
Every classification must be tied to evidence:
- Victim statements
- Bank records
- Device fingerprint analysis
- IP address logs
- Corporate ownership documents
- Email correspondence
- Etc.

**Why:** Without evidence, the SAR filing may be rejected

### 2. Exclusions Are As Important As Inclusions
Document why transactions are NOT proceeds:
- Legitimate creditor payments
- Normal business expenses
- Transactions to entities not controlled by suspect

**Why:** Demonstrates you considered all transactions and made reasoned decisions

### 3. User Must Confirm at Each Phase
Cannot skip confirmation gates:
- Confirms you reviewed the data
- Confirms you made the decision consciously
- Creates audit trail of your confirmations

**Why:** Proves you exercised professional judgment

### 4. Complete Audit Trail
Every decision is timestamped and documented:
- What you decided
- When you decided it
- Why you decided it
- What evidence supports it

**Why:** Legal defensibility and regulatory compliance

---

## Testing and Validation

To test this workflow:

1. **Mock Investigation Scenario**:
   - Create test data with fraud transactions
   - Walk through all 7 phases
   - Verify user controls work correctly
   - Check audit trail completeness

2. **Edge Cases**:
   - Unknown quantum (estimates required)
   - Mixed currencies
   - Very deep layering chains (10+ hops)
   - Complex exclusion scenarios

3. **Integration Testing**:
   - Verify works with `get-transaction-history`
   - Check output feeds into SAR Section 2.2
   - Test with different database schemas

4. **Compliance Review**:
   - Have MLRO review workflow
   - Verify meets SAR requirements
   - Check audit trail sufficiency

---

## Version History

- **v1.0** (2024-01-20): Initial release
  - 7-phase user-controlled workflow
  - Multiple confirmation gates
  - Complete audit trail
  - Evidence-based classification
  - SAR-compliant output

---

## Support and Questions

For questions about this tool:
1. Review the full YAML description (917 lines of guidance)
2. Check SAR Requirements documentation
3. Consult with MLRO/compliance team
4. Test with mock scenarios before live use

**Remember:** This tool gives you control, but you must exercise professional judgment based on evidence. The tool cannot make decisions for you - that's by design.
