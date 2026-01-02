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

	// Debug: Print the events found
	t.Logf("Found %d events:", len(eventsJson.Events))
	for i, event := range eventsJson.Events {
		t.Logf("Event %d: %s (%s to %s, %d days)", i+1, event.Name, event.StartDate, event.EndDate, event.Duration)
	}

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

func TestRealStructure(t *testing.T) {
	file, err := static.TestFiles.Open("testdata/events/real_structure.html")
	if err != nil {
		t.Fatalf("file opening error: %s", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("File reading error: %s", err)
	}

	eventsJson, err := TibiaEventsCalendarImpl(string(data), "https://www.tibia.com/news/?subtopic=eventcalendar&calendarmonth=2&calendaryear=2026", 2, 2026)
	if err != nil {
		t.Fatal(err)
	}

	assert := assert.New(t)
	information := eventsJson.Information

	assert.Equal("https://www.tibia.com/news/?subtopic=eventcalendar&calendarmonth=2&calendaryear=2026", information.TibiaURLs[0])

	// Debug: Print the events found
	t.Logf("Found %d events:", len(eventsJson.Events))
	for i, event := range eventsJson.Events {
		t.Logf("Event %d: %s (%s to %s, %d days)", i+1, event.Name, event.StartDate, event.EndDate, event.Duration)
	}

	// Check that we have events
	assert.Greater(len(eventsJson.Events), 0, "Should have at least one event")

	// Check month and year are set correctly
	assert.Equal(2, eventsJson.Month, "Month should be February (2)")
	assert.Equal(2026, eventsJson.Year, "Year should be 2026")

	// Check for expected events
	eventNames := make(map[string]bool)
	for _, event := range eventsJson.Events {
		eventNames[event.Name] = true
	}

	// Should contain these events from the real structure
	expectedEvents := []string{"The First Dragon", "Full Moon", "Valentine's Day", "Last Creep Standing", "A Piece of Cake"}
	for _, expected := range expectedEvents {
		assert.True(eventNames[expected], "Should contain event: %s", expected)
	}
}

func TestRealTibiaStructure(t *testing.T) {
	file, err := static.TestFiles.Open("testdata/events/real_tibia_structure.html")
	if err != nil {
		t.Fatalf("file opening error: %s", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("File reading error: %s", err)
	}

	eventsJson, err := TibiaEventsCalendarImpl(string(data), "https://www.tibia.com/news/?subtopic=eventcalendar&calendarmonth=2&calendaryear=2026", 2, 2026)
	if err != nil {
		t.Fatal(err)
	}

	assert := assert.New(t)
	information := eventsJson.Information

	assert.Equal("https://www.tibia.com/news/?subtopic=eventcalendar&calendarmonth=2&calendaryear=2026", information.TibiaURLs[0])

	// Debug: Print the events found
	t.Logf("Found %d events:", len(eventsJson.Events))
	for i, event := range eventsJson.Events {
		t.Logf("Event %d: %s (%s to %s, %d days)", i+1, event.Name, event.StartDate, event.EndDate, event.Duration)
	}

	// Check that we have events
	assert.Greater(len(eventsJson.Events), 0, "Should have at least one event")

	// Check month and year are set correctly
	assert.Equal(2, eventsJson.Month, "Month should be February (2)")
	assert.Equal(2026, eventsJson.Year, "Year should be 2026")

	// Check for expected events from the real Tibia structure
	eventNames := make(map[string]bool)
	for _, event := range eventsJson.Events {
		eventNames[event.Name] = true
	}

	// Should contain these events from the real Tibia structure
	expectedEvents := []string{"The First Dragon", "Full Moon", "Valentine's Day", "Last Creep Standing", "A Piece of Cake"}
	for _, expected := range expectedEvents {
		assert.True(eventNames[expected], "Should contain event: %s", expected)
	}
}