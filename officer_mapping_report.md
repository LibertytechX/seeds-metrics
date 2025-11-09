# Officer Mapping Report: Django → Seeds Metrics

**Date**: 2025-11-09  
**Purpose**: Map loan officer IDs from verticals.tsv (Django customer_user_id) to Seeds Metrics officer_id  

---

## Executive Summary

Successfully mapped **100% of the 550 matching loan officers** from the Django database to the Seeds Metrics database using email-based matching.

### Key Findings

| Metric | Count | Percentage |
|--------|-------|------------|
| **Django customer_user_ids processed** | 550 | 100% |
| **✅ Matched in Seeds Metrics (by email)** | **550** | **100%** |
| **❌ Not found in Seeds Metrics** | 0 | 0% |

---

## 🎯 Critical Discovery: Different ID Systems

### The Problem

The `verticals.tsv` file uses **Django `customer_user_id`** values (e.g., 10001, 10002, 5425), but the Seeds Metrics database uses **different `officer_id`** values (e.g., 4886, 678, 1101).

### The Solution

**Email-based mapping** successfully bridges the two ID systems:

| Django customer_user_id | Django Email | Seeds officer_id | Match |
|------------------------|--------------|------------------|-------|
| 10001 | abosedeayeni159@gmail.com | 4886 | ✅ |
| 3513 | josephineayodele2@gmail.com | 678 | ✅ |
| 5425 | nofisatogundelee@gmail.com | 1101 | ✅ |
| 5430 | onuohachukwukere@gmail.com | 1057 | ✅ |

---

## 📊 Mapping Statistics

### ID System Comparison

| System | ID Field | Total Records | ID Range |
|--------|----------|---------------|----------|
| **Django** | `customer_user_id` | 6,799 | 95 - 10,000+ |
| **Seeds Metrics** | `officer_id` | 6,800 | 1 - 6,800 |
| **verticals.tsv** | `loan_officer_id` | 558 | 1032 - 10,329 |

### Matching Results

- **558 IDs in verticals.tsv**
- **550 IDs exist in Django** (98.57% match)
- **550 IDs mapped to Seeds Metrics** (100% of valid IDs)
- **8 IDs in TSV not in Django** (1.43% - likely deleted users)

---

## 📋 Sample Mappings

Here are 20 sample mappings showing the relationship between Django and Seeds Metrics:

| Django ID | Django Email | Seeds ID | Seeds Email | Status |
|-----------|--------------|----------|-------------|--------|
| 3513 | josephineayodele2@gmail.com | 678 | josephineayodele2@gmail.com | ✅ MATCHED |
| 3797 | silvianwakuna@gmail.com | 510 | silvianwakuna@gmail.com | ✅ MATCHED |
| 5425 | nofisatogundelee@gmail.com | 1101 | nofisatogundelee@gmail.com | ✅ MATCHED |
| 5430 | onuohachukwukere@gmail.com | 1057 | onuohachukwukere@gmail.com | ✅ MATCHED |
| 5431 | adeyemigem@gmail.com | 1043 | adeyemigem@gmail.com | ✅ MATCHED |
| 5434 | ademolaro@gmail.com | 1049 | ademolaro@gmail.com | ✅ MATCHED |
| 5450 | obamo2012@gmail.com | 1053 | obamo2012@gmail.com | ✅ MATCHED |
| 5452 | adebowaleseun1234@gmail.com | 1055 | adebowaleseun1234@gmail.com | ✅ MATCHED |
| 5664 | taiwoolarenwaju6@gmail.com | 1100 | taiwoolarenwaju6@gmail.com | ✅ MATCHED |
| 5673 | innocentomoses@yahoo.com | 1099 | innocentomoses@yahoo.com | ✅ MATCHED |
| 5674 | ifycharles99@gmail.com | 1098 | ifycharles99@gmail.com | ✅ MATCHED |
| 5676 | infoadinoyi2@gmail.com | 1097 | infoadinoyi2@gmail.com | ✅ MATCHED |
| 5678 | adeniyimaryoluwafunmilayo@gmail.com | 1103 | adeniyimaryoluwafunmilayo@gmail.com | ✅ MATCHED |
| 5702 | ogunleyedeborah22@gmail.com | 1107 | ogunleyedeborah22@gmail.com | ✅ MATCHED |
| 5732 | michisonline@gmail.com | 1129 | michisonline@gmail.com | ✅ MATCHED |
| 5733 | ayodejipreciouslawal@gmail.com | 1141 | ayodejipreciouslawal@gmail.com | ✅ MATCHED |
| 5734 | belloharbey@gmail.com | 1127 | belloharbey@gmail.com | ✅ MATCHED |
| 5737 | lamidmuinat@gmail.com | 1142 | lamidmuinat@gmail.com | ✅ MATCHED |
| 6285 | boluwatifer35@gmail.com | 1183 | boluwatifer35@gmail.com | ✅ MATCHED |
| 6286 | dahmolyn03@gmail.com | 1176 | dahmolyn03@gmail.com | ✅ MATCHED |

---

## 🗂️ Generated Files

All mapping data has been saved to the following files:

### Primary Mapping File
- **`/tmp/officer_mapping.txt`** - Complete mapping (551 records including header)
  - Format: `customer_user_id|django_email|seeds_officer_id|seeds_email|match_status`
  - Use this file for updating the loans table with vertical/branch information

### Supporting Files
- **`/tmp/django_users_emails.txt`** - Django customer_user_id|email (550 records)
- **`/tmp/seeds_officers_emails.txt`** - Seeds officer_id|email (6,778 records)
- **`/tmp/matching_ids.txt`** - 550 matching customer_user_ids from Django
- **`/tmp/in_both_systems.txt`** - 99 IDs that exist in both systems (by ID, not email)
- **`/tmp/missing_from_seeds.txt`** - 451 Django IDs not in Seeds (by ID comparison)

---

## 🔍 Key Insights

### 1. ID Mismatch Between Systems

**Direct ID comparison**: Only 18% match (99 out of 550)
- Django customer_user_id: 10001
- Seeds officer_id: 4886
- **These are DIFFERENT IDs for the same person!**

**Email-based comparison**: 100% match (550 out of 550)
- Both systems use the same email addresses
- Email is the reliable linking field

### 2. Why the ID Systems Differ

The Seeds Metrics database appears to have been populated independently from Django, resulting in:
- Different auto-increment sequences
- Different ID assignment logic
- No direct 1:1 ID mapping

### 3. Email as the Universal Key

Email addresses are **consistent across both systems** and serve as the reliable identifier for mapping officers.

---

## ✅ Data Quality Assessment

### Overall Quality: **EXCELLENT** ✅

- **100% email match rate** - All valid Django officers exist in Seeds Metrics
- **98.57% TSV accuracy** - Only 8 invalid IDs in verticals.tsv
- **Reliable mapping** - Email-based linking is robust and accurate

---

## 🎯 Next Steps for Vertical/Branch Updates

Now that we have a complete mapping, here's how to update the loans table:

### Step 1: Create Enhanced Mapping File

Combine the officer mapping with verticals.tsv data:
```
seeds_officer_id | vertical | branch | supervisor | region
```

### Step 2: Update Officers Table

Add vertical, branch, and supervisor columns to the `officers` table in Seeds Metrics.

### Step 3: Update Loans Table

Use the mapping to update all loans with their officer's vertical/branch information.

### Recommended Approach

```sql
-- Example: Update officers table with vertical information
UPDATE officers o
SET 
    vertical = v.vertical,
    branch = v.branch,
    supervisor = v.supervisor
FROM verticals_mapping v
WHERE o.officer_id = v.seeds_officer_id;

-- Then update loans table
UPDATE loans l
SET
    vertical = o.vertical,
    branch = o.branch,
    supervisor = o.supervisor
FROM officers o
WHERE l.officer_id = o.officer_id;
```

---

## 📝 Important Notes

### For Future Data Syncs

1. **Always use email for matching** - Don't rely on customer_user_id matching officer_id
2. **Maintain the mapping file** - Keep `/tmp/officer_mapping.txt` updated
3. **Validate before updates** - Always check match rates before bulk updates

### Data Consistency

- The 8 IDs in verticals.tsv that don't exist in Django should be investigated
- Consider removing or correcting these entries in verticals.tsv
- See `id_comparison_report.md` for details on the 8 missing IDs

---

## 🎉 Conclusion

**Status**: ✅ **SUCCESS** - Complete officer mapping achieved!

We have successfully:
1. ✅ Identified that Django and Seeds Metrics use different ID systems
2. ✅ Discovered that email is the reliable linking field
3. ✅ Mapped 100% of valid officers (550 out of 550)
4. ✅ Created comprehensive mapping files for the next phase

**The mapping is ready to use for updating loans with vertical/branch information.**

---

## 📊 Summary Statistics

```
Total IDs in verticals.tsv:              558
Valid IDs in Django:                     550  (98.57%)
Mapped to Seeds Metrics:                 550  (100% of valid)
Ready for vertical/branch updates:       550  officers
Estimated loans affected:                ~17,929 loans
```

---

*Report generated by: analyze_officer_mapping.sh*  
*Mapping date: 2025-11-09*  
*Mapping method: Email-based matching*

