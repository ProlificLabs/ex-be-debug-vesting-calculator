package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMonthsBetweenCalculation(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected int
	}{
		{
			name:     "Same date",
			start:    time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "One month exactly",
			start:    time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 2, 15, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "Partial month rounds down",
			start:    time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 2, 10, 0, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "12 months exactly",
			start:    time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: 12,
		},
		{
			name:     "24 months exactly",
			start:    time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: 24,
		},
		{
			name:     "Leap year handling",
			start:    time.Date(2020, 2, 29, 0, 0, 0, 0, time.UTC),
			end:      time.Date(2021, 2, 28, 0, 0, 0, 0, time.UTC),
			expected: 11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monthsBetween(tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("monthsBetween(%v, %v) = %d, want %d",
					tt.start.Format("2006-01-02"),
					tt.end.Format("2006-01-02"),
					result, tt.expected)
			}
		})
	}
}

func TestLinearVestingCalculations(t *testing.T) {
	service := NewVestingService()

	tests := []struct {
		name           string
		employee       Employee
		asOfDate       time.Time
		expectedVested int
		description    string
	}{
		{
			name: "Before cliff",
			employee: Employee{
				ID:         "emp1",
				Name:       "Pre-cliff Employee",
				StartDate:  time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: 48000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "linear",
				},
			},
			asOfDate:       time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
			expectedVested: 0,
			description:    "10 months employed, still in cliff period",
		},
		{
			name: "Exactly at cliff",
			employee: Employee{
				ID:         "emp2",
				Name:       "At-cliff Employee",
				StartDate:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: 48000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "linear",
				},
			},
			asOfDate:       time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedVested: 0,
			description:    "12 months employed, just reached cliff",
		},
		{
			name: "One year after cliff",
			employee: Employee{
				ID:         "emp3",
				Name:       "Mid-vesting Employee",
				StartDate:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: 48000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "linear",
				},
			},
			asOfDate:       time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedVested: 12000,
			description:    "24 months employed, 12 months vesting after cliff",
		},
		{
			name: "Fully vested",
			employee: Employee{
				ID:         "emp4",
				Name:       "Fully-vested Employee",
				StartDate:  time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: 48000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "linear",
				},
			},
			asOfDate:       time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
			expectedVested: 48000,
			description:    "Over 48 months employed, fully vested",
		},
		{
			name: "Partial month vesting",
			employee: Employee{
				ID:         "emp5",
				Name:       "Partial-month Employee",
				StartDate:  time.Date(2021, 1, 15, 0, 0, 0, 0, time.UTC),
				TotalUnits: 36000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 36,
					VestingType:   "linear",
				},
			},
			asOfDate:       time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC),
			expectedVested: 11000,
			description:    "23 months employed (partial), 11 months vesting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ProcessBatch([]Employee{tt.employee}, tt.asOfDate)
			if err != nil {
				t.Fatalf("ProcessBatch failed: %v", err)
			}

			result, exists := service.GetResult(tt.employee.ID)
			if !exists {
				t.Fatal("Result not found in cache")
			}

			if result.VestedUnits != tt.expectedVested {
				t.Errorf("%s: expected %d vested units, got %d. %s",
					tt.name, tt.expectedVested, result.VestedUnits, tt.description)
			}

			// Verify unvested units
			expectedUnvested := tt.employee.TotalUnits - tt.expectedVested
			if result.UnvestedUnits != expectedUnvested {
				t.Errorf("%s: expected %d unvested units, got %d",
					tt.name, expectedUnvested, result.UnvestedUnits)
			}
		})
	}
}

func TestBackloadedVestingCalculations(t *testing.T) {
	service := NewVestingService()

	tests := []struct {
		name           string
		employee       Employee
		asOfDate       time.Time
		expectedVested int
		description    string
	}{
		{
			name: "End of year 1",
			employee: Employee{
				ID:         "back1",
				Name:       "Year 1 Employee",
				StartDate:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: 40000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "backloaded",
				},
			},
			asOfDate:       time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedVested: 4000,
			description:    "24 months: should have 10% vested",
		},
		{
			name: "Mid year 2",
			employee: Employee{
				ID:         "back2",
				Name:       "Mid Year 2 Employee",
				StartDate:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: 40000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "backloaded",
				},
			},
			asOfDate:       time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC),
			expectedVested: 8000,
			description:    "30 months: should have 10% + 10% (half of year 2's 20%)",
		},
		{
			name: "End of year 3",
			employee: Employee{
				ID:         "back3",
				Name:       "Year 3 Employee",
				StartDate:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: 40000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "backloaded",
				},
			},
			asOfDate:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedVested: 24000,
			description:    "48 months: should have 10% + 20% + 30% = 60%",
		},
		{
			name: "Fully vested",
			employee: Employee{
				ID:         "back4",
				Name:       "Fully Vested Employee",
				StartDate:  time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: 40000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "backloaded",
				},
			},
			asOfDate:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedVested: 40000,
			description:    "60+ months: should be 100% vested",
		},
		{
			name: "Partial year vesting",
			employee: Employee{
				ID:         "back5",
				Name:       "Partial Year Employee",
				StartDate:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: 50000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "backloaded",
				},
			},
			asOfDate:       time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
			expectedVested: 12500,
			description:    "33 months: 10% + 20% + 25% (9/12 of year 3's 30%)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ProcessBatch([]Employee{tt.employee}, tt.asOfDate)
			if err != nil {
				t.Fatalf("ProcessBatch failed: %v", err)
			}

			result, exists := service.GetResult(tt.employee.ID)
			if !exists {
				t.Fatal("Result not found in cache")
			}

			if result.VestedUnits != tt.expectedVested {
				t.Errorf("%s: expected %d vested units, got %d. %s",
					tt.name, tt.expectedVested, result.VestedUnits, tt.description)
			}
		})
	}
}

func TestConcurrentProcessingAndRaceConditions(t *testing.T) {
	service := NewVestingService()

	// Create employees with predictable vesting
	employees := make([]Employee, 100)
	for i := 0; i < 100; i++ {
		employees[i] = Employee{
			ID:         fmt.Sprintf("concurrent_emp_%d", i),
			Name:       fmt.Sprintf("Employee %d", i),
			StartDate:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			TotalUnits: 10000, // Fixed amount for easier verification
			Schedule: VestingSchedule{
				CliffMonths:   12,
				VestingMonths: 48,
				VestingType:   "linear",
			},
		}
	}

	asOfDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	// Run multiple concurrent batches to test for race conditions
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for run := 0; run < 10; run++ {
		wg.Add(1)
		go func(runNum int) {
			defer wg.Done()

			// Each run processes the same employees
			err := service.ProcessBatch(employees, asOfDate)
			if err != nil {
				errors <- fmt.Errorf("run %d: %v", runNum, err)
			}
		}(run)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Fatal(err)
	}

	// Verify all results are correct and consistent
	for i := 0; i < 100; i++ {
		employeeID := fmt.Sprintf("concurrent_emp_%d", i)
		result, exists := service.GetResult(employeeID)
		if !exists {
			t.Errorf("Result not found for employee %s", employeeID)
			continue
		}

		// All employees should have same vesting: 12 months out of 36 post-cliff months
		expectedVested := 3333 // 10000 * 12/36
		if result.VestedUnits != expectedVested {
			t.Errorf("Employee %s: expected %d vested units, got %d",
				employeeID, expectedVested, result.VestedUnits)
		}
	}
}

func TestEdgeCasesAndValidation(t *testing.T) {
	service := NewVestingService()

	tests := []struct {
		name        string
		employee    Employee
		asOfDate    time.Time
		shouldError bool
		description string
	}{
		{
			name: "Zero total units",
			employee: Employee{
				ID:         "zero_units",
				Name:       "Zero Units Employee",
				StartDate:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: 0,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "linear",
				},
			},
			asOfDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			shouldError: true,
			description: "Should error on zero units",
		},
		{
			name: "Negative total units",
			employee: Employee{
				ID:         "neg_units",
				Name:       "Negative Units Employee",
				StartDate:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: -1000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "linear",
				},
			},
			asOfDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			shouldError: true,
			description: "Should error on negative units",
		},
		{
			name: "Future start date",
			employee: Employee{
				ID:         "future_start",
				Name:       "Future Start Employee",
				StartDate:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				TotalUnits: 10000,
				Schedule: VestingSchedule{
					CliffMonths:   12,
					VestingMonths: 48,
					VestingType:   "linear",
				},
			},
			asOfDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			shouldError: false,
			description: "Future start date should result in 0 vested",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ProcessBatch([]Employee{tt.employee}, tt.asOfDate)

			if tt.shouldError && err == nil {
				t.Errorf("%s: expected error but got none. %s", tt.name, tt.description)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("%s: unexpected error: %v. %s", tt.name, err, tt.description)
			}
		})
	}
}

func TestCacheOperations(t *testing.T) {
	service := NewVestingService()

	employee := Employee{
		ID:         "cache_test",
		Name:       "Cache Test Employee",
		StartDate:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		TotalUnits: 10000,
		Schedule: VestingSchedule{
			CliffMonths:   12,
			VestingMonths: 48,
			VestingType:   "linear",
		},
	}

	asOfDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	// Process employee
	err := service.ProcessBatch([]Employee{employee}, asOfDate)
	if err != nil {
		t.Fatalf("ProcessBatch failed: %v", err)
	}

	// Verify result exists
	result1, exists := service.GetResult(employee.ID)
	if !exists {
		t.Fatal("Result not found after processing")
	}

	// Clear cache
	service.ClearCache()

	// Verify result no longer exists
	_, exists = service.GetResult(employee.ID)
	if exists {
		t.Error("Result still exists after clearing cache")
	}

	// Process again with different date
	newDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	err = service.ProcessBatch([]Employee{employee}, newDate)
	if err != nil {
		t.Fatalf("Second ProcessBatch failed: %v", err)
	}

	// Verify new result
	result2, exists := service.GetResult(employee.ID)
	if !exists {
		t.Fatal("Result not found after second processing")
	}

	// Results should be different
	if result1.VestedUnits >= result2.VestedUnits {
		t.Error("Expected more vesting after additional year")
	}
}

func TestBatchResultsRetrieval(t *testing.T) {
	service := NewVestingService()

	// Create and process multiple employees
	employees := []Employee{
		{
			ID:         "batch1",
			Name:       "Batch Employee 1",
			StartDate:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			TotalUnits: 10000,
			Schedule: VestingSchedule{
				CliffMonths:   12,
				VestingMonths: 48,
				VestingType:   "linear",
			},
		},
		{
			ID:         "batch2",
			Name:       "Batch Employee 2",
			StartDate:  time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
			TotalUnits: 20000,
			Schedule: VestingSchedule{
				CliffMonths:   12,
				VestingMonths: 48,
				VestingType:   "linear",
			},
		},
		{
			ID:         "batch3",
			Name:       "Batch Employee 3",
			StartDate:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			TotalUnits: 15000,
			Schedule: VestingSchedule{
				CliffMonths:   12,
				VestingMonths: 48,
				VestingType:   "backloaded",
			},
		},
	}

	asOfDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	err := service.ProcessBatch(employees, asOfDate)
	if err != nil {
		t.Fatalf("ProcessBatch failed: %v", err)
	}

	// Test successful batch retrieval
	employeeIDs := []string{"batch1", "batch2", "batch3"}
	results, err := service.GetBatchResults(employeeIDs)
	if err != nil {
		t.Fatalf("GetBatchResults failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Test missing employee
	missingIDs := []string{"batch1", "missing_employee"}
	_, err = service.GetBatchResults(missingIDs)
	if err == nil {
		t.Error("Expected error for missing employee, got none")
	}
}

func TestScheduleValidation(t *testing.T) {
	tests := []struct {
		name        string
		schedule    VestingSchedule
		shouldError bool
	}{
		{
			name: "Valid linear schedule",
			schedule: VestingSchedule{
				CliffMonths:   12,
				VestingMonths: 48,
				VestingType:   "linear",
			},
			shouldError: false,
		},
		{
			name: "Valid backloaded schedule",
			schedule: VestingSchedule{
				CliffMonths:   12,
				VestingMonths: 48,
				VestingType:   "backloaded",
			},
			shouldError: false,
		},
		{
			name: "Negative cliff months",
			schedule: VestingSchedule{
				CliffMonths:   -1,
				VestingMonths: 48,
				VestingType:   "linear",
			},
			shouldError: true,
		},
		{
			name: "Vesting months less than cliff",
			schedule: VestingSchedule{
				CliffMonths:   12,
				VestingMonths: 12,
				VestingType:   "linear",
			},
			shouldError: true,
		},
		{
			name: "Invalid vesting type",
			schedule: VestingSchedule{
				CliffMonths:   12,
				VestingMonths: 48,
				VestingType:   "exponential",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSchedule(tt.schedule)
			if tt.shouldError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}