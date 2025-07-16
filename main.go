package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	// Create a new vesting service
	service := NewVestingService()

	// Example employees with different vesting schedules
	employees := []Employee{
		{
			ID:         "emp001",
			Name:       "Alice Johnson",
			StartDate:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			TotalUnits: 48000,
			Schedule: VestingSchedule{
				CliffMonths:   12,
				VestingMonths: 48,
				VestingType:   "linear",
			},
		},
		{
			ID:         "emp002",
			Name:       "Bob Smith",
			StartDate:  time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
			TotalUnits: 60000,
			Schedule: VestingSchedule{
				CliffMonths:   12,
				VestingMonths: 48,
				VestingType:   "backloaded",
			},
		},
		{
			ID:         "emp003",
			Name:       "Carol Davis",
			StartDate:  time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
			TotalUnits: 40000,
			Schedule: VestingSchedule{
				CliffMonths:   6,
				VestingMonths: 36,
				VestingType:   "linear",
			},
		},
	}

	// Calculate vesting as of today
	asOfDate := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)

	fmt.Println("=== Pulley Vesting Calculator ===")
	fmt.Printf("Calculating vesting as of: %s\n\n", asOfDate.Format("2006-01-02"))

	// Process all employees
	err := service.ProcessBatch(employees, asOfDate)
	if err != nil {
		log.Fatalf("Error processing batch: %v", err)
	}

	// Display results
	for _, emp := range employees {
		result, exists := service.GetResult(emp.ID)
		if !exists {
			fmt.Printf("ERROR: No result found for %s\n", emp.Name)
			continue
		}

		fmt.Printf("Employee: %s\n", emp.Name)
		fmt.Printf("  Start Date: %s\n", emp.StartDate.Format("2006-01-02"))
		fmt.Printf("  Total Units: %d\n", emp.TotalUnits)
		fmt.Printf("  Vesting Type: %s (%d month cliff, %d months total)\n", 
			emp.Schedule.VestingType, emp.Schedule.CliffMonths, emp.Schedule.VestingMonths)
		fmt.Printf("  Vested Units: %d (%.1f%%)\n", 
			result.VestedUnits, float64(result.VestedUnits)/float64(emp.TotalUnits)*100)
		fmt.Printf("  Unvested Units: %d\n", result.UnvestedUnits)
		
		if !result.NextVestDate.IsZero() && result.VestedUnits < emp.TotalUnits {
			fmt.Printf("  Next Vest Date: %s\n", result.NextVestDate.Format("2006-01-02"))
		} else if result.VestedUnits >= emp.TotalUnits {
			fmt.Printf("  Status: Fully Vested\n")
		}
		fmt.Println()
	}

	// Example of batch retrieval
	fmt.Println("=== Batch Retrieval Example ===")
	employeeIDs := []string{"emp001", "emp002"}
	results, err := service.GetBatchResults(employeeIDs)
	if err != nil {
		log.Printf("Error retrieving batch results: %v", err)
	} else {
		fmt.Printf("Successfully retrieved %d results\n", len(results))
		for id, result := range results {
			fmt.Printf("  %s: %d vested units\n", id, result.VestedUnits)
		}
	}
}