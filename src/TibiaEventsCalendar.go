package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Event represents a single event from the calendar
type Event struct {
	Name      string `json:"name"`       // The name of the event.
	StartDate string `json:"start_date"` // The start date of the event (YYYY-MM-DD format).
	EndDate   string `json:"end_date"`   // The end date of the event (YYYY-MM-DD format).
	Duration  int    `json:"duration"`   // Number of days the event runs.
}

// EventsCalendarResponse is the response structure for events calendar
type EventsCalendarResponse struct {
	Month       int         `json:"month"`       // Calendar month displayed.
	Year        int         `json:"year"`        // Calendar year displayed.
	Events      []Event     `json:"events"`      // List of events in the calendar.
	Information Information `json:"information"` // API information.
}

var (
	// Regex patterns for parsing event information
	eventDateRegex = regexp.MustCompile(`(\d{1,2})(?:-(\d{1,2}))?`)
)

// TibiaEventsCalendarImpl parses the event calendar HTML and returns structured data
func TibiaEventsCalendarImpl(BoxContentHTML string, url string, month int, year int) (EventsCalendarResponse, error) {
	// Loading HTML data into goquery
	ReaderHTML, err := goquery.NewDocumentFromReader(strings.NewReader(BoxContentHTML))
	if err != nil {
		return EventsCalendarResponse{}, fmt.Errorf("[error] TibiaEventsCalendarImpl failed at goquery.NewDocumentFromReader, err: %s", err)
	}

	var EventsData []Event
	var calendarMonth, calendarYear int
	var insideError error

	// If month and year not provided, try to extract from the page
	if month == 0 || year == 0 {
		// Try to extract current month/year from the calendar header
		ReaderHTML.Find(".TableContainer select[name='calendarmonth'] option[selected]").Each(func(index int, s *goquery.Selection) {
			monthValue, exists := s.Attr("value")
			if exists {
				calendarMonth = TibiaDataStringToInteger(monthValue)
			}
		})

		ReaderHTML.Find(".TableContainer select[name='calendaryear'] option[selected]").Each(func(index int, s *goquery.Selection) {
			yearValue, exists := s.Attr("value")
			if exists {
				calendarYear = TibiaDataStringToInteger(yearValue)
			}
		})

		// Use current date if still not found
		if calendarMonth == 0 || calendarYear == 0 {
			now := time.Now()
			if calendarMonth == 0 {
				calendarMonth = int(now.Month())
			}
			if calendarYear == 0 {
				calendarYear = now.Year()
			}
		}
	} else {
		calendarMonth = month
		calendarYear = year
	}

	// Map to track unique events
	eventMap := make(map[string]*Event)

	// Find the calendar table
	ReaderHTML.Find("table.Table3").EachWithBreak(func(tableIndex int, table *goquery.Selection) bool {
		// Look for calendar cells with events
		table.Find("td").EachWithBreak(func(cellIndex int, cell *goquery.Selection) bool {
			cellText := strings.TrimSpace(cell.Text())

			// Skip empty cells or cells with just day numbers
			if cellText == "" || regexp.MustCompile(`^\d{1,2}$`).MatchString(cellText) {
				return true
			}

			// Check if cell contains an event (has a div with event class or special formatting)
			cell.Find("div").Each(func(divIndex int, div *goquery.Selection) {
				eventText := strings.TrimSpace(div.Text())

				// Skip if it's just a day number
				if eventText == "" || regexp.MustCompile(`^\d{1,2}$`).MatchString(eventText) {
					return
				}

				// Extract the day number from the cell
				dayStr := ""
				cell.Contents().Each(func(i int, s *goquery.Selection) {
					text := strings.TrimSpace(s.Text())
					if regexp.MustCompile(`^\d{1,2}$`).MatchString(text) {
						dayStr = text
						return
					}
				})

				// If we found an event name and have a day
				if eventText != "" && dayStr != "" {
					day := TibiaDataStringToInteger(dayStr)
					if day > 0 {
						// Create or update event
						if existingEvent, exists := eventMap[eventText]; exists {
							// Update end date if this day is later
							endDay := TibiaDataStringToInteger(strings.Split(existingEvent.EndDate, "-")[2])
							if day > endDay {
								existingEvent.EndDate = fmt.Sprintf("%04d-%02d-%02d", calendarYear, calendarMonth, day)
							}
						} else {
							// Create new event
							event := Event{
								Name:      eventText,
								StartDate: fmt.Sprintf("%04d-%02d-%02d", calendarYear, calendarMonth, day),
								EndDate:   fmt.Sprintf("%04d-%02d-%02d", calendarYear, calendarMonth, day),
							}
							eventMap[eventText] = &event
						}
					}
				}
			})

			return true
		})

		return true
	})

	// Alternative parsing method - look for event list if calendar table parsing didn't work
	if len(eventMap) == 0 {
		// Look for events in different format (list view or special event containers)
		ReaderHTML.Find(".InnerTableContainer").Each(func(index int, container *goquery.Selection) {
			container.Find("tr").Each(func(rowIndex int, row *goquery.Selection) {
				rowText := strings.TrimSpace(row.Text())

				// Look for patterns like "Event Name (Jan 1 - Jan 15)"
				if strings.Contains(rowText, "-") || strings.Contains(rowText, "to") {
					// Try to extract event information
					parts := strings.Split(rowText, "(")
					if len(parts) >= 2 {
						eventName := strings.TrimSpace(parts[0])
						datePart := strings.TrimSpace(strings.TrimSuffix(parts[1], ")"))

						// Parse date range
						if eventName != "" && datePart != "" {
							event := Event{
								Name:      eventName,
								StartDate: fmt.Sprintf("%04d-%02d-01", calendarYear, calendarMonth),
								EndDate:   fmt.Sprintf("%04d-%02d-01", calendarYear, calendarMonth),
							}
							eventMap[eventName] = &event
						}
					}
				}
			})
		})
	}

	// Convert map to slice and calculate durations
	for _, event := range eventMap {
		// Calculate duration
		startTime, err1 := time.Parse("2006-01-02", event.StartDate)
		endTime, err2 := time.Parse("2006-01-02", event.EndDate)

		if err1 == nil && err2 == nil {
			duration := endTime.Sub(startTime)
			event.Duration = int(duration.Hours()/24) + 1 // +1 because we include both start and end days
		} else {
			event.Duration = 1 // Default to 1 day if parsing fails
		}

		EventsData = append(EventsData, *event)
	}

	if insideError != nil {
		return EventsCalendarResponse{}, insideError
	}

	// Build the response
	return EventsCalendarResponse{
		Month:  calendarMonth,
		Year:   calendarYear,
		Events: EventsData,
		Information: Information{
			APIDetails: TibiaDataAPIDetails,
			Timestamp:  TibiaDataDatetime(""),
			TibiaURLs:  []string{url},
			Status: Status{
				HTTPCode: http.StatusOK,
			},
		},
	}, nil
}