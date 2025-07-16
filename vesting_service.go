package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type VestingService struct {
	cache *VestingCache
	mu    sync.Mutex
}

func NewVestingService() *VestingService {
	return &VestingService{
		cache: NewVestingCache(),
	}
}

// ProcessBatch calculates vesting for multiple employees concurrently
func (vs *VestingService) ProcessBatch(employees []Employee, asOfDate time.Time) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(employees))

	for _, emp := range employees {
		wg.Add(1)
		go func(employee Employee) {
			defer wg.Done()
			result, err := vs.calculateVesting(employee, asOfDate)
			if err != nil {
				errors <- err
				return
			}

			// Store result in cache
			vs.cache.results[employee.ID] = result
		}(emp)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

// calculateVesting calculates vested units for a single employee
func (vs *VestingService) calculateVesting(employee Employee, asOfDate time.Time) (VestingResult, error) {
	if employee.TotalUnits <= 0 {
		return VestingResult{}, fmt.Errorf("invalid total units: %d", employee.TotalUnits)
	}

	monthsEmployed := monthsBetween(employee.StartDate, asOfDate)

	// Check if still in cliff period
	if monthsEmployed < employee.Schedule.CliffMonths {
		return VestingResult{
			EmployeeID:    employee.ID,
			VestedUnits:   0,
			UnvestedUnits: employee.TotalUnits,
			NextVestDate:  addMonths(employee.StartDate, employee.Schedule.CliffMonths),
			AsOfDate:      asOfDate,
		}, nil
	}

	var vestedUnits int

	if employee.Schedule.VestingType == "linear" {
		// Linear vesting: equal amounts each month after cliff
		monthsVested := monthsEmployed - employee.Schedule.CliffMonths
		if monthsVested > employee.Schedule.VestingMonths - employee.Schedule.CliffMonths {
			monthsVested = employee.Schedule.VestingMonths - employee.Schedule.CliffMonths
		}

		vestingMonthsAfterCliff := employee.Schedule.VestingMonths - employee.Schedule.CliffMonths
		unitsPerMonth := float64(employee.TotalUnits) / float64(vestingMonthsAfterCliff)
		vestedUnits = int(unitsPerMonth * float64(monthsVested))

	} else if employee.Schedule.VestingType == "backloaded" {
		// Backloaded vesting: 10% year 1, 20% year 2, 30% year 3, 40% year 4
		yearsVested := (monthsEmployed - employee.Schedule.CliffMonths) / 12

		percentages := []float64{0.1, 0.2, 0.3, 0.4}
		totalPercent := 0.0

		for i := 0; i <= yearsVested && i < len(percentages); i++ {
			totalPercent += percentages[i]
		}

		// Add partial year vesting for current year
		monthsInCurrentYear := (monthsEmployed - employee.Schedule.CliffMonths) % 12
		if yearsVested < len(percentages) && monthsInCurrentYear > 0 {
			currentYearPercent := percentages[yearsVested] * (float64(monthsInCurrentYear) / 12.0)
			totalPercent += currentYearPercent
		}

		vestedUnits = int(math.Floor(float64(employee.TotalUnits) * totalPercent))
	}

	// Calculate next vest date
	var nextVestDate time.Time
	if vestedUnits < employee.TotalUnits {
		if employee.Schedule.VestingType == "linear" {
			nextVestDate = addMonths(asOfDate, 1)
		} else {
			// For backloaded, next vest is at the next month
			nextVestDate = addMonths(asOfDate, 1)
		}
	}

	return VestingResult{
		EmployeeID:    employee.ID,
		VestedUnits:   vestedUnits,
		UnvestedUnits: employee.TotalUnits - vestedUnits,
		NextVestDate:  nextVestDate,
		AsOfDate:      asOfDate,
	}, nil
}

// GetResult retrieves a vesting result from cache
func (vs *VestingService) GetResult(employeeID string) (VestingResult, bool) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	result, exists := vs.cache.results[employeeID]
	return result, exists
}

// ClearCache clears all cached results
func (vs *VestingService) ClearCache() {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.cache = NewVestingCache()
}

// Helper functions
func monthsBetween(start, end time.Time) int {
	months := 0
	month := start.Month()
	year := start.Year()

	for start.Before(end) {
		start = start.AddDate(0, 1, 0)
		months++

		if start.Month() != month {
			month = start.Month()
		}
		if start.Year() != year {
			year = start.Year()
		}
	}

	return months
}

func addMonths(t time.Time, months int) time.Time {
	return t.AddDate(0, months, 0)
}

// GetBatchResults returns all results for a list of employee IDs
func (vs *VestingService) GetBatchResults(employeeIDs []string) (map[string]VestingResult, error) {
	results := make(map[string]VestingResult)

	for _, id := range employeeIDs {
		if result, exists := vs.GetResult(id); exists {
			results[id] = result
		} else {
			return nil, fmt.Errorf("result not found for employee %s", id)
		}
	}

	return results, nil
}

// ValidateSchedule ensures vesting schedule parameters are valid
func ValidateSchedule(schedule VestingSchedule) error {
	if schedule.CliffMonths < 0 {
		return fmt.Errorf("cliff months cannot be negative")
	}
	if schedule.VestingMonths <= schedule.CliffMonths {
		return fmt.Errorf("total vesting months must be greater than cliff months")
	}
	if schedule.VestingType != "linear" && schedule.VestingType != "backloaded" {
		return fmt.Errorf("invalid vesting type: %s", schedule.VestingType)
	}
	return nil
}
