package matching

import (
	"math"
	"strings"
	"time"

	"reconciliation-service/internal/models"
)

const (
	// Match confidence thresholds
	PerfectMatchConfidence = 1.00
	HighMatchConfidence    = 0.95
	MediumMatchConfidence  = 0.80
	LowMatchConfidence     = 0.60

	// Amount difference tolerance (in percentage)
	AmountTolerancePercent = 0.01 // 1%

	// Date difference tolerance (in days)
	DateToleranceDays = 3
)

type MatchResult struct {
	Type              string  // one_to_one, one_to_many
	Confidence        float64 // 0.00 to 1.00
	BankTransaction   *models.BankTransaction
	AccountingEntries []*models.AccountingEntry
	AmountDifference  float64
	MatchCriteria     []string
}

type MatchesResult struct {
	Type             string  // one_to_one, one_to_many
	Confidence       float64 // 0.00 to 1.00
	BankTransaction  string
	AccountingEntry  string
	AmountDifference float64
	MatchCriteria    []string
}

type UnmatchResult struct {
	BankTransactions  string
	AccountingEntries []string
}

type MatchEngine struct {
	bankTransactions  []*models.BankTransaction
	accountingEntries []*models.AccountingEntry
}

func NewMatchEngine() *MatchEngine {
	return &MatchEngine{}
}

func (m *MatchEngine) SetData(bankTransactions []*models.BankTransaction, accountingEntries []*models.AccountingEntry) {
	m.bankTransactions = bankTransactions
	m.accountingEntries = accountingEntries
}

func (m *MatchEngine) ProcessMatches() ([]*MatchResult, error) {
	var results []*MatchResult

	processedBankIDs := make(map[int64]bool)
	processedAccountingIDs := make(map[int64]bool)

	for _, bt := range m.bankTransactions {
		if processedBankIDs[bt.ID] {
			continue
		}

		for _, ae := range m.accountingEntries {
			if processedAccountingIDs[ae.ID] {
				continue
			}

			if result := m.checkOneToOneMatch(bt, ae); result != nil && result.Confidence == PerfectMatchConfidence {
				results = append(results, result)
				processedBankIDs[bt.ID] = true
				processedAccountingIDs[ae.ID] = true
				break
			}
		}
	}

	for _, bt := range m.bankTransactions {
		if processedBankIDs[bt.ID] {
			continue
		}

		if result := m.findOneToManyMatch(bt, processedAccountingIDs); result != nil {
			results = append(results, result)
			processedBankIDs[bt.ID] = true
			for _, ae := range result.AccountingEntries {
				processedAccountingIDs[ae.ID] = true
			}
		}
	}

	for _, bt := range m.bankTransactions {
		if processedBankIDs[bt.ID] {
			continue
		}

		var bestMatch *MatchResult
		var bestConfidence float64

		for _, ae := range m.accountingEntries {
			if processedAccountingIDs[ae.ID] {
				continue
			}

			if result := m.checkOneToOneMatch(bt, ae); result != nil && result.Confidence > bestConfidence {
				bestMatch = result
				bestConfidence = result.Confidence
			}
		}

		if bestMatch != nil && bestMatch.Confidence >= LowMatchConfidence {
			results = append(results, bestMatch)
			processedBankIDs[bt.ID] = true
			processedAccountingIDs[bestMatch.AccountingEntries[0].ID] = true
		}
	}

	return results, nil
}

func (m *MatchEngine) checkOneToOneMatch(bt *models.BankTransaction, ae *models.AccountingEntry) *MatchResult {
	var matchCriteria []string
	var confidence float64

	amountDiff := math.Abs(bt.Amount - ae.Amount)
	amountTolerance := bt.Amount * AmountTolerancePercent

	if amountDiff == 0 {
		matchCriteria = append(matchCriteria, "amount")
		confidence += 0.4
	} else if amountDiff <= amountTolerance {
		matchCriteria = append(matchCriteria, "amount")
		confidence += 0.3
	} else {
		return nil // Amount difference too large
	}

	btDate, _ := time.Parse("2006-01-02", bt.TransactionDate)
	aeDate, _ := time.Parse("2006-01-02", ae.EntryDate)
	dateDiff := math.Abs(float64(btDate.Sub(aeDate).Hours() / 24))

	if dateDiff == 0 {
		matchCriteria = append(matchCriteria, "date")
		confidence += 0.3
	} else if dateDiff <= float64(DateToleranceDays) {
		matchCriteria = append(matchCriteria, "date")
		confidence += 0.2
	}

	if bt.ReferenceNumber != "" && ae.InvoiceNumber != "" {
		if bt.ReferenceNumber == ae.InvoiceNumber {
			matchCriteria = append(matchCriteria, "reference")
			confidence += 0.3
		} else {
			confidence = 0
		}
	}

	if confidence >= LowMatchConfidence {
		return &MatchResult{
			Type:              models.MappingOneToOne,
			Confidence:        confidence,
			BankTransaction:   bt,
			AccountingEntries: []*models.AccountingEntry{ae},
			AmountDifference:  amountDiff,
			MatchCriteria:     matchCriteria,
		}
	}

	return nil
}

func (m *MatchEngine) findOneToManyMatch(bt *models.BankTransaction, processedIDs map[int64]bool) *MatchResult {
	var bestMatch *MatchResult
	var minDifference float64 = bt.Amount // Start with the full amount as the difference

	combinations := m.findPossibleEntryCombinations(bt, bt.Amount, processedIDs)

	for _, entries := range combinations {
		var totalAmount float64
		for _, ae := range entries {
			totalAmount += ae.Amount
		}

		difference := math.Abs(bt.Amount - totalAmount)
		if difference < minDifference {
			minDifference = difference

			confidence := m.calculateOneToManyConfidence(bt, entries, difference)

			var matchCriteria []string
			matchCriteria = append(matchCriteria, "amount")

			btDate, _ := time.Parse("2006-01-02", bt.TransactionDate)
			var maxDateDiff float64
			for _, ae := range entries {
				aeDate, _ := time.Parse("2006-01-02", ae.EntryDate)
				dateDiff := math.Abs(float64(btDate.Sub(aeDate).Hours() / 24))
				if dateDiff > maxDateDiff {
					maxDateDiff = dateDiff
				}
			}

			if maxDateDiff <= float64(DateToleranceDays) {
				matchCriteria = append(matchCriteria, "date")
			}

			if bt.ReferenceNumber != "" {
				for _, ae := range entries {
					if ae.InvoiceNumber != "" && strings.Contains(ae.InvoiceNumber, bt.ReferenceNumber) {
						matchCriteria = append(matchCriteria, "reference")
						break
					}
				}
			}

			if confidence >= MediumMatchConfidence {
				bestMatch = &MatchResult{
					Type:              models.MappingOneToMany,
					Confidence:        confidence,
					BankTransaction:   bt,
					AccountingEntries: entries,
					AmountDifference:  difference,
					MatchCriteria:     matchCriteria,
				}
			}
		}
	}

	return bestMatch
}

func (m *MatchEngine) findPossibleEntryCombinations(bt *models.BankTransaction, targetAmount float64, processedIDs map[int64]bool) [][]*models.AccountingEntry {
	var result [][]*models.AccountingEntry
	var candidates []*models.AccountingEntry

	for _, ae := range m.accountingEntries {
		if !processedIDs[ae.ID] && ae.Amount <= targetAmount {
			if bt.ReferenceNumber != "" && ae.InvoiceNumber != "" &&
				strings.Contains(ae.InvoiceNumber, bt.ReferenceNumber) {
				candidates = append([]*models.AccountingEntry{ae}, candidates...)
			}
		}
	}

	for i := 1; i <= 3; i++ {
		m.findCombinations(candidates, i, targetAmount, nil, &result)
	}

	return result
}

func (m *MatchEngine) findCombinations(candidates []*models.AccountingEntry, size int, targetAmount float64, current []*models.AccountingEntry, result *[][]*models.AccountingEntry) {
	if size == 0 {
		var sum float64
		for _, ae := range current {
			sum += ae.Amount
		}

		if math.Abs(targetAmount-sum) <= (targetAmount * AmountTolerancePercent) {
			combination := make([]*models.AccountingEntry, len(current))
			copy(combination, current)
			*result = append(*result, combination)
		}
		return
	}

	if len(candidates) < size {
		return
	}

	m.findCombinations(candidates[1:], size-1, targetAmount, append(current, candidates[0]), result)
	m.findCombinations(candidates[1:], size, targetAmount, current, result)
}

func (m *MatchEngine) calculateOneToManyConfidence(bt *models.BankTransaction, entries []*models.AccountingEntry, amountDiff float64) float64 {
	var confidence float64 = 0.7 // Base confidence for matching sum

	if amountDiff == 0 {
		confidence += 0.2
	} else if amountDiff <= (bt.Amount * AmountTolerancePercent) {
		confidence += 0.1
	}

	btDate, _ := time.Parse("2006-01-02", bt.TransactionDate)
	var maxDateDiff float64
	for _, ae := range entries {
		aeDate, _ := time.Parse("2006-01-02", ae.EntryDate)
		dateDiff := math.Abs(float64(btDate.Sub(aeDate).Hours() / 24))
		if dateDiff > maxDateDiff {
			maxDateDiff = dateDiff
		}
	}

	if maxDateDiff <= float64(DateToleranceDays) {
		confidence += 0.1
	}

	if bt.ReferenceNumber != "" {
		matchCount := 0
		for _, ae := range entries {
			if ae.InvoiceNumber != "" && strings.Contains(ae.InvoiceNumber, bt.ReferenceNumber) {
				matchCount++
			}
		}
		if matchCount > 0 {
			confidence += 0.1 * float64(matchCount) / float64(len(entries))
		}
	}

	if confidence > HighMatchConfidence {
		confidence = HighMatchConfidence
	}

	return confidence
}
