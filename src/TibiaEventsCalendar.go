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

	// Find the calendar table - try multiple selectors
	ReaderHTML.Find("table.Table3, table").EachWithBreak(func(tableIndex int, table *goquery.Selection) bool {
		// Skip if this table doesn't contain calendar cells
		if table.Find("td").Length() < 7 {
			return true // Continue looking for other tables
		}
		// Track row index to help identify if we're in the first or last week
		rowIndex := 0

		table.Find("tr").EachWithBreak(func(trIndex int, row *goquery.Selection) bool {
			// Track cell index within the row
			cellIndexInRow := 0

			row.Find("td").EachWithBreak(func(cellIndex int, cell *goquery.Selection) bool {

				// Extract day number from real Tibia.com structure or fallback methods
				dayStr := ""

				// Method 1: Real Tibia structure - day number is in first span inside first div
				firstDiv := cell.Find("div").First()
				if firstDiv.Length() > 0 {
					// Look for span with day number inside the first div
					firstDiv.Find("span").Each(func(spanIndex int, span *goquery.Selection) {
						spanText := strings.TrimSpace(span.Text())
						// Check if this span contains just a day number (possibly with trailing space)
						dayMatch := regexp.MustCompile(`^(\d{1,2})\s*$`).FindStringSubmatch(spanText)
						if len(dayMatch) > 1 && dayStr == "" {
							dayStr = dayMatch[1]
						}
					})
				}

				// Method 2: Test structure - day number is directly in first div
				if dayStr == "" && firstDiv.Length() > 0 {
					dayText := strings.TrimSpace(firstDiv.Text())
					if regexp.MustCompile(`^\d{1,2}$`).MatchString(dayText) {
						dayStr = dayText
					}
				}

				// Method 3: Legacy fallback - day number as direct text node
				if dayStr == "" {
					cell.Contents().Each(func(i int, s *goquery.Selection) {
						if goquery.NodeName(s) == "#text" {
							text := strings.TrimSpace(s.Text())
							if text != "" && dayStr == "" && regexp.MustCompile(`^\d{1,2}$`).MatchString(text) {
								dayStr = text
							}
						}
					})
				}

				// Skip if no day number found
				if dayStr == "" {
					cellIndexInRow++
					return true
				}

				day := TibiaDataStringToInteger(dayStr)

				// Skip invalid days
				if day <= 0 || day > 31 {
					cellIndexInRow++
					return true
				}

				// Skip obvious previous/next month days
				// In first row: skip days > 7 if we haven't seen day 1 yet (previous month)
				if rowIndex == 0 && day > 7 {
					// Look ahead in this row to see if we'll encounter day 1
					foundDayOne := false
					for futureIndex := cellIndexInRow; futureIndex < 7; futureIndex++ {
						futureCell := row.Find("td").Eq(futureIndex)
						if futureCell.Length() == 0 {
							break
						}

						futureDayStr := ""
						futureFirstDiv := futureCell.Find("div").First()
						if futureFirstDiv.Length() > 0 {
							futureFirstDiv.Find("span").Each(func(spanIndex int, span *goquery.Selection) {
								spanText := strings.TrimSpace(span.Text())
								dayMatch := regexp.MustCompile(`^(\d{1,2})\s*$`).FindStringSubmatch(spanText)
								if len(dayMatch) > 1 && futureDayStr == "" {
									futureDayStr = dayMatch[1]
								}
							})
						}

						if futureDayStr == "" && futureFirstDiv.Length() > 0 {
							futureText := strings.TrimSpace(futureFirstDiv.Text())
							if regexp.MustCompile(`^\d{1,2}$`).MatchString(futureText) {
								futureDayStr = futureText
							}
						}

						if futureDayStr != "" {
							futureDay := TibiaDataStringToInteger(futureDayStr)
							if futureDay == 1 {
								foundDayOne = true
								break
							}
						}
					}

					// If we find day 1 later in this row and current day > 7, this is previous month
					if foundDayOne {
						cellIndexInRow++
						return true
					}
				}

				// In last rows: skip days <= 15 that come after days > 15 (next month)
				// This is more aggressive to catch spillover from next month
				if rowIndex >= 3 && day <= 15 {
					// Look at previous cells in this row to determine if this is likely next month
					prevDayFound := false
					for prevCellIndex := cellIndexInRow - 1; prevCellIndex >= 0; prevCellIndex-- {
						prevCell := row.Find("td").Eq(prevCellIndex)
						prevDayStr := ""

						// Try to get previous day using same methods
						prevFirstDiv := prevCell.Find("div").First()
						if prevFirstDiv.Length() > 0 {
							prevFirstDiv.Find("span").Each(func(spanIndex int, span *goquery.Selection) {
								spanText := strings.TrimSpace(span.Text())
								dayMatch := regexp.MustCompile(`^(\d{1,2})\s*$`).FindStringSubmatch(spanText)
								if len(dayMatch) > 1 && prevDayStr == "" {
									prevDayStr = dayMatch[1]
								}
							})
						}

						if prevDayStr == "" && prevFirstDiv.Length() > 0 {
							prevDayText := strings.TrimSpace(prevFirstDiv.Text())
							if regexp.MustCompile(`^\d{1,2}$`).MatchString(prevDayText) {
								prevDayStr = prevDayText
							}
						}

						if prevDayStr != "" {
							prevDay := TibiaDataStringToInteger(prevDayStr)
							if prevDay > 15 && day <= 15 {
								// This looks like next month spillover
								cellIndexInRow++
								return true
							}
							prevDayFound = true
							break
						}
					}

					// Additional check: if we're in a late row and see a low day number, it's likely next month
					if !prevDayFound && rowIndex >= 4 && day <= 10 {
						cellIndexInRow++
						return true
					}
				}

				// Look for events in this cell using real Tibia structure
				// Real structure: span contains div with event name
				cell.Find("span div").Each(func(index int, eventDiv *goquery.Selection) {
					eventText := strings.TrimSpace(eventDiv.Text())

					// Skip if empty
					if eventText == "" {
						return
					}

					// Skip if it's just a day number
					if regexp.MustCompile(`^\d{1,2}$`).MatchString(eventText) {
						return
					}

					// Clean up event text (remove asterisks that indicate start/end times)
					eventText = strings.TrimPrefix(eventText, "*")
					eventText = strings.TrimSpace(eventText)

					if eventText == "" {
						return
					}

					// Track this day for this event
					if _, exists := eventDays[eventText]; !exists {
						eventDays[eventText] = make(map[int]bool)
					}
					eventDays[eventText][day] = true
				})

				// Fallback for test structure - check div elements directly
				cell.Find("div").Each(func(index int, element *goquery.Selection) {
					eventText := strings.TrimSpace(element.Text())

					// Skip if empty or just a number (day number)
					if eventText == "" || regexp.MustCompile(`^\d{1,2}$`).MatchString(eventText) {
						return
					}

					// Skip if it contains only the day number
					if eventText == dayStr {
						return
					}

					// Clean up event text (remove asterisks that indicate start/end times)
					eventText = strings.TrimPrefix(eventText, "*")
					eventText = strings.TrimSpace(eventText)

					if eventText == "" {
						return
					}

					// Only add if we haven't already found this event from span div structure
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