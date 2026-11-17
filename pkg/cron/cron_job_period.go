package cron

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
)

// CronJobPeriod represents the cron job period
type CronJobPeriod interface {
	GetInterval() time.Duration
	ToJobDefinition() gocron.JobDefinition
}

// CronJobIntervalPeriod represents the period of execution at intervals
type CronJobIntervalPeriod struct {
	Interval time.Duration
}

// CronJobFixedHourPeriod represents the period of execution at fixed hour
type CronJobFixedHourPeriod struct {
	Hour uint32
}

// CronJobEvery15MinutesPeriod represents the period of execution at every 15 minutes
type CronJobEvery15MinutesPeriod struct {
	Second uint32
}

// CronJobFixedTimePeriod represents the period of execution at fixed time
type CronJobFixedTimePeriod struct {
	Time time.Time
}

// CronJobWeeklyPeriod represents the period of execution at fixed weekday and time
type CronJobWeeklyPeriod struct {
	Weekday time.Weekday
	Hour    uint32
	Minute  uint32
}

// GetInterval returns the interval time of the period of CronJobIntervalPeriod
func (p CronJobIntervalPeriod) GetInterval() time.Duration {
	return p.Interval
}

// ToJobDefinition returns the gocron job definition of the period of CronJobIntervalPeriod
func (p CronJobIntervalPeriod) ToJobDefinition() gocron.JobDefinition {
	return gocron.DurationJob(p.Interval)
}

// GetInterval returns the interval time of the period of CronJobFixedHourPeriod
func (p CronJobFixedHourPeriod) GetInterval() time.Duration {
	return 24 * time.Hour
}

// ToJobDefinition returns the gocron job definition of the period of CronJobFixedHourPeriod
func (p CronJobFixedHourPeriod) ToJobDefinition() gocron.JobDefinition {
	return gocron.DailyJob(
		1,
		gocron.NewAtTimes(
			gocron.NewAtTime(uint(p.Hour), 0, 0),
		),
	)
}

// GetInterval returns the interval time of the period of CronJobEvery15MinutesPeriod
func (p CronJobEvery15MinutesPeriod) GetInterval() time.Duration {
	return 15 * time.Minute
}

// ToJobDefinition returns the gocron job definition of the period of CronJobEvery15MinutesPeriod
func (p CronJobEvery15MinutesPeriod) ToJobDefinition() gocron.JobDefinition {
	return gocron.CronJob(fmt.Sprintf("%d */15 * * * *", p.Second), true)
}

// GetInterval returns the interval time of the period of CronJobFixedTimePeriod
func (p CronJobFixedTimePeriod) GetInterval() time.Duration {
	return 0
}

// ToJobDefinition returns the gocron job definition of the period of CronJobFixedTimePeriod
func (p CronJobFixedTimePeriod) ToJobDefinition() gocron.JobDefinition {
	return gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(p.Time))
}

// GetInterval returns the interval time of the period of CronJobWeeklyPeriod
func (p CronJobWeeklyPeriod) GetInterval() time.Duration {
	return 7 * 24 * time.Hour
}

// ToJobDefinition returns the gocron job definition of the period of CronJobWeeklyPeriod
func (p CronJobWeeklyPeriod) ToJobDefinition() gocron.JobDefinition {
	return gocron.CronJob(fmt.Sprintf("0 %d %d * * %d", p.Minute, p.Hour, p.Weekday), true)
}
