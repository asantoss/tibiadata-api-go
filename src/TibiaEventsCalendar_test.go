package main

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tibiadata/tibiadata-api-go/src/static"
)

func TestEventsCalendar(t *testing.T) {
	file, err := static.TestFiles.Open("testdata/events/eventcalendar.html")
	if err != nil {
		t.Fatalf("file opening error: %s", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("File reading error: %s", err)
	}

	eventsJson, err := TibiaEventsCalendarImpl(string(data), "https://www.tibia.com/news/?subtopic=eventcalendar", 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	assert := assert.New(t)
	information := eventsJson.Information

	assert.Equal("https://www.tibia.com/news/?subtopic=eventcalendar", information.TibiaURLs[0])

	// Check that we have events
	assert.Greater(len(eventsJson.Events), 0, "Should have at least one event")

	// Check the structure of the first event if it exists
	if len(eventsJson.Events) > 0 {
		firstEvent := eventsJson.Events[0]
		assert.NotEmpty(firstEvent.Name, "Event name should not be empty")
		assert.NotEmpty(firstEvent.StartDate, "Start date should not be empty")
		assert.NotEmpty(firstEvent.EndDate, "End date should not be empty")
		assert.GreaterOrEqual(firstEvent.Duration, 1, "Duration should be at least 1 day")
	}

	// Check month and year are set
	assert.Greater(eventsJson.Month, 0, "Month should be set")
	assert.Greater(eventsJson.Year, 0, "Year should be set")
	assert.LessOrEqual(eventsJson.Month, 12, "Month should be valid")
	assert.GreaterOrEqual(eventsJson.Year, 2020, "Year should be reasonable")
}

func TestEventsCalendarWithMonthYear(t *testing.T) {
	file, err := static.TestFiles.Open("testdata/events/eventcalendar_march2026.html")
	if err != nil {
		// Skip test if file doesn't exist yet
		t.Skip("Test data file not yet created")
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("File reading error: %s", err)
	}

	eventsJson, err := TibiaEventsCalendarImpl(string(data), "https://www.tibia.com/news/?subtopic=eventcalendar&calendarmonth=3&calendaryear=2026", 3, 2026)
	if err != nil {
		t.Fatal(err)
	}

	assert := assert.New(t)

	// Check specific month and year
	assert.Equal(3, eventsJson.Month, "Month should be March (3)")
	assert.Equal(2026, eventsJson.Year, "Year should be 2026")
}