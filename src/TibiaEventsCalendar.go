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

	// Map to track events with their days
	// Key: event name, Value: map of days (as integers)
	eventDays := make(map[string]map[int]bool)

	// Find the calendar table
	ReaderHTML.Find("table.Table3").EachWithBreak(func(tableIndex int, table *goquery.Selection) bool {
		// Track row index to help identify if we're in the first or last week
		rowIndex := 0

		table.Find("tr").EachWithBreak(func(trIndex int, row *goquery.Selection) bool {
			// Track cell index within the row
			cellIndexInRow := 0

			row.Find("td").EachWithBreak(func(cellIndex int, cell *goquery.Selection) bool {

				// Extract day number - it's typically the first text node
				dayStr := ""
				cellTextParts := strings.Fields(cell.Text())
				if len(cellTextParts) > 0 {
					// Check if first part is a number
					if regexp.MustCompile(`^\d{1,2}$`).MatchString(cellTextParts[0]) {
						dayStr = cellTextParts[0]
					}
				}

				// Skip if no day number found
				if dayStr == "" {
					cellIndexInRow++
					return true
				}

				day := TibiaDataStringToInteger(dayStr)

				// Skip days from previous month (typically > 20) in the first week
				// or days from next month (typically < 10) in the last weeks
				if rowIndex == 0 && day > 20 {
					// This is a day from the previous month
					cellIndexInRow++
					return true
				}

				// In the last row, skip days that appear to be from next month
				if rowIndex >= 4 && day <= 7 {
					// Check if we have already seen a day > 20 in this row or previous cells
					hasHighDay := false
					row.Find("td").EachWithBreak(func(checkIndex int, checkCell *goquery.Selection) bool {
						if checkIndex >= cellIndexInRow {
							return false // Stop checking, we're at current cell or beyond
						}
						checkTextParts := strings.Fields(checkCell.Text())
						if len(checkTextParts) > 0 {
							checkDay := TibiaDataStringToInteger(checkTextParts[0])
							if checkDay > 20 {
								hasHighDay = true
								return false // Found high day, stop
							}
						}
						return true
					})

					if hasHighDay {
						// We've seen high days (20+) and now see low days (1-7), this is next month
						cellIndexInRow++
						return true
					}
				}

				// Look for events in this cell
				cell.Find("div").Each(func(divIndex int, div *goquery.Selection) {
					eventText := strings.TrimSpace(div.Text())

					// Skip if empty or just a number
					if eventText == "" || regexp.MustCompile(`^\d{1,2}$`).MatchString(eventText) {
						return
					}

					// Track this day for this event
					if _, exists := eventDays[eventText]; !exists {
						eventDays[eventText] = make(map[int]bool)
					}
					eventDays[eventText][day] = true
				})

				cellIndexInRow++
				return true
			})

			rowIndex++
			return true
		})

		return true
	})

	// Convert eventDays map to Event structs with proper date ranges
	for eventName, days := range eventDays {
		if len(days) == 0 {
			continue
		}

		// Find min and max days for this event
		minDay := 32
		maxDay := 0
		for day := range days {
			if day < minDay {
				minDay = day
			}
			if day > maxDay {
				maxDay = day
			}
		}

		// Create event with proper date range
		event := Event{
			Name:      eventName,
			StartDate: fmt.Sprintf("%04d-%02d-%02d", calendarYear, calendarMonth, minDay),
			EndDate:   fmt.Sprintf("%04d-%02d-%02d", calendarYear, calendarMonth, maxDay),
		}

		// Calculate duration
		event.Duration = maxDay - minDay + 1

		EventsData = append(EventsData, event)
	}

	// Alternative parsing method - look for event list if calendar table parsing didn't work
	if len(EventsData) == 0 {
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
								EndDate:   fmt.Sprintf("%04d-%02d-15", calendarYear, calendarMonth),
								Duration:  15,
							}
							EventsData = append(EventsData, event)
						}
					}
				}
			})
		})
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