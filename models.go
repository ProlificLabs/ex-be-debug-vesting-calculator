package main

import (
	"time"
)

type Employee struct {
	ID         string
	Name       string
	StartDate  time.Time
	TotalUnits int
	Schedule   VestingSchedule
}

type VestingSchedule struct {
	CliffMonths   int
	VestingMonths int
	VestingType   string // "linear" or "backloaded"
}

type VestingResult struct {
	EmployeeID    string
	VestedUnits   int
	UnvestedUnits int
	NextVestDate  time.Time
	AsOfDate      time.Time
}

type VestingCache struct {
	results map[string]VestingResult
}

func NewVestingCache() *VestingCache {
	return &VestingCache{
		results: make(map[string]VestingResult),
	}
}
