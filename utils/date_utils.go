package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// DateSearchResult represents the result of parsing search text for date patterns
type DateSearchResult struct {
	IsDate     bool         // Whether the text was parsed as a date
	IsDayOnly  bool         // Whether it's just a day number (e.g., "18")
	Date       *time.Time   // The parsed date (if IsDate is true and single date)
	Dates      []time.Time  // Multiple parsed dates (if IsDate is true and multiple dates)
	Day        *int         // The day number (if IsDayOnly is true)
	SearchText string       // The original search text
}

// ParseSearchTextForDate intelligently parses search text to detect dates, partial dates, or day-only numbers
// This function can be reused across all modules for consistent date search behavior
func ParseSearchTextForDate(searchText string) DateSearchResult {
	result := DateSearchResult{
		SearchText: searchText,
	}

	if searchText == "" {
		return result
	}

	// Try to parse common date formats
	dateFormats := []string{
		"2 Jan 2006",      // "18 Aug 2025"
		"2 January 2006",  // "18 August 2025"
		"2006-01-02",      // "2025-08-18"
		"02/01/2006",      // "18/08/2025"
		"01/02/2006",      // "08/18/2006"
		"2006/01/02",      // "2025/08/18"
		"Jan 2, 2006",     // "Aug 18, 2025"
		"January 2, 2006", // "August 18, 2025"
		"2-Jan-2006",      // "18-Aug-2025"
		"2/Jan/2006",      // "18/Aug/2025"
	}

	// First try exact date formats
	for _, format := range dateFormats {
		if parsed, err := time.Parse(format, searchText); err == nil {
			result.IsDate = true
			result.Date = &parsed
			fmt.Printf("DEBUG: Parsed date '%s' as: %s\n", searchText, parsed.Format("2006-01-02"))
			return result
		}
	}

	// If no exact match, try to detect partial date patterns
	parts := strings.Fields(searchText)
	if len(parts) == 2 {
		// Try to parse as "18 Aug" or "Aug 18"
		dayStr := parts[0]
		monthStr := parts[1]

		// Try "18 Aug" format
		if day, err := strconv.Atoi(dayStr); err == nil && day >= 1 && day <= 31 {
			// Convert month name to time.Month using a map
			monthMap := map[string]time.Month{
				"Jan": time.January, "Feb": time.February, "Mar": time.March, "Apr": time.April,
				"May": time.May, "Jun": time.June, "Jul": time.July, "Aug": time.August,
				"Sep": time.September, "Oct": time.October, "Nov": time.November, "Dec": time.December,
			}
			if month, exists := monthMap[monthStr]; exists {
				// Use current year for partial dates
				currentYear := time.Now().Year()
				parsed := time.Date(currentYear, month, day, 0, 0, 0, 0, time.UTC)
				result.IsDate = true
				result.Date = &parsed
				fmt.Printf("DEBUG: Detected partial date '%s' as day %d, month %s, using year %d\n",
					searchText, day, month.String(), currentYear)
				return result
			}
			
			// Handle incomplete month abbreviations (e.g., "18 A" -> treat as "18 Aug")
			if len(monthStr) >= 1 {
				// Try to find all months that start with the given letter(s)
				monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
				var matchingDates []time.Time
				currentYear := time.Now().Year()
				
				// Convert month name to time.Month using a map
				monthMap := map[string]time.Month{
					"Jan": time.January, "Feb": time.February, "Mar": time.March, "Apr": time.April,
					"May": time.May, "Jun": time.June, "Jul": time.July, "Aug": time.August,
					"Sep": time.September, "Oct": time.October, "Nov": time.November, "Dec": time.December,
				}
				
				for _, monthName := range monthNames {
					if strings.HasPrefix(strings.ToLower(monthName), strings.ToLower(monthStr)) {
						if month, exists := monthMap[monthName]; exists {
							parsed := time.Date(currentYear, month, day, 0, 0, 0, 0, time.UTC)
							matchingDates = append(matchingDates, parsed)
						}
					}
				}
				
				if len(matchingDates) > 0 {
					result.IsDate = true
					if len(matchingDates) == 1 {
						// Single match - use the Date field for backward compatibility
						result.Date = &matchingDates[0]
						fmt.Printf("DEBUG: Detected incomplete month '%s' as '%s', parsed date: day %d, month %s, using year %d\n",
							monthStr, monthNames[0], day, monthNames[0], currentYear)
					} else {
						// Multiple matches - use the Dates field
						result.Dates = matchingDates
						fmt.Printf("DEBUG: Detected incomplete month '%s' with %d matches: %v\n",
							monthStr, len(matchingDates), matchingDates)
					}
					return result
				}
			}
		}

		// Try "Aug 18" format if first attempt failed
		monthMap := map[string]time.Month{
			"Jan": time.January, "Feb": time.February, "Mar": time.March, "Apr": time.April,
			"May": time.May, "Jun": time.June, "Jul": time.July, "Aug": time.August,
			"Sep": time.September, "Oct": time.October, "Nov": time.November, "Dec": time.December,
		}
		if month, exists := monthMap[monthStr]; exists {
			if day, err := strconv.Atoi(dayStr); err == nil && day >= 1 && day <= 31 {
				currentYear := time.Now().Year()
				parsed := time.Date(currentYear, month, day, 0, 0, 0, 0, time.UTC)
				result.IsDate = true
				result.Date = &parsed
				fmt.Printf("DEBUG: Detected partial date '%s' as month %s, day %d, using year %d\n",
					searchText, month.String(), day, currentYear)
				return result
			}
		}
	}

	// If not a date, detect numeric-only day-of-month input (e.g., "18")
	// Also handle cases like "18a" where day is followed by incomplete month
	onlyDigits := true
	firstNonDigitIndex := -1
	for i, r := range searchText {
		if !unicode.IsDigit(r) {
			onlyDigits = false
			firstNonDigitIndex = i
			break
		}
	}
	
	if onlyDigits {
		if d, err := strconv.Atoi(searchText); err == nil && d >= 1 && d <= 31 {
			result.IsDayOnly = true
			result.Day = &d
			fmt.Printf("DEBUG: Detected numeric day-of-month: %d\n", d)
			return result
		}
	} else if firstNonDigitIndex > 0 {
		// Handle cases like "18a", "25j", etc.
		dayStr := searchText[:firstNonDigitIndex]
		monthPrefix := searchText[firstNonDigitIndex:]
		
		if day, err := strconv.Atoi(dayStr); err == nil && day >= 1 && day <= 31 {
			// Try to find a month that starts with the given prefix
			monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
			for _, monthName := range monthNames {
				if strings.HasPrefix(strings.ToLower(monthName), strings.ToLower(monthPrefix)) {
					// Use current year for partial dates
					currentYear := time.Now().Year()
					// Convert month name to time.Month using a map
					monthMap := map[string]time.Month{
						"Jan": time.January, "Feb": time.February, "Mar": time.March, "Apr": time.April,
						"May": time.May, "Jun": time.June, "Jul": time.July, "Aug": time.August,
						"Sep": time.September, "Oct": time.October, "Nov": time.November, "Dec": time.December,
					}
					if month, exists := monthMap[monthName]; exists {
						parsed := time.Date(currentYear, month, day, 0, 0, 0, 0, time.UTC)
						result.IsDate = true
						result.Date = &parsed
						fmt.Printf("DEBUG: Detected combined date '%s' as day %d, month %s, using year %d\n",
							searchText, day, monthName, currentYear)
						return result
					}
				}
			}
		}
	}

	// If we get here, it's not a date or day-only
	fmt.Printf("DEBUG: Could not parse '%s' as a date, treating as text search\n", searchText)
	return result
}

// BuildDateSearchQuery builds the appropriate WHERE clause for date searching
// This function can be reused across all modules for consistent date search behavior
func BuildDateSearchQuery(baseQuery interface{}, searchText string, dateResult DateSearchResult, tableAlias string) (string, []interface{}) {
	return BuildDateSearchQueryWithColumns(baseQuery, searchText, dateResult, tableAlias, "title", "statement", "name")
}

// BuildDateSearchQueryWithColumns builds the appropriate WHERE clause for date searching with custom column names
// This function allows specifying which columns to search in for different table structures
func BuildDateSearchQueryWithColumns(baseQuery interface{}, searchText string, dateResult DateSearchResult, tableAlias string, titleCol, statementCol, nameCol string) (string, []interface{}) {
	if dateResult.IsDate {
		// Date search: search for records created on the specific date(s)
		var dateConditions []string
		var args []interface{}
		
		// Add text search conditions
		args = append(args, "%"+searchText+"%", "%"+searchText+"%", "%"+searchText+"%") // title, statement, name
		
		// Handle single date vs multiple dates
		if dateResult.Date != nil {
			// Single date
			startOfDay := time.Date(dateResult.Date.Year(), dateResult.Date.Month(), dateResult.Date.Day(), 0, 0, 0, 0, dateResult.Date.Location())
			endOfDay := startOfDay.Add(24 * time.Hour)
			dateConditions = append(dateConditions, fmt.Sprintf("(%s.created_at >= ? AND %s.created_at < ?)", tableAlias, tableAlias))
			args = append(args, startOfDay, endOfDay)
		} else if len(dateResult.Dates) > 0 {
			// Multiple dates - create OR conditions for each date
			for _, date := range dateResult.Dates {
				startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
				endOfDay := startOfDay.Add(24 * time.Hour)
				dateConditions = append(dateConditions, fmt.Sprintf("(%s.created_at >= ? AND %s.created_at < ?)", tableAlias, tableAlias))
				args = append(args, startOfDay, endOfDay)
			}
		}
		
		// Build the WHERE clause
		whereClause := fmt.Sprintf("LOWER(%s.%s) LIKE ? OR LOWER(%s.%s) LIKE ? OR LOWER(%s.%s) LIKE ?", 
			tableAlias, titleCol, tableAlias, statementCol, tableAlias, nameCol)
		
		// Add date conditions if any
		if len(dateConditions) > 0 {
			whereClause += " OR " + strings.Join(dateConditions, " OR ")
		}
		
		return whereClause, args
	} else if dateResult.IsDayOnly {
		// Day-only search: include day-of-month filter
		whereClause := fmt.Sprintf("LOWER(%s.%s) LIKE ? OR LOWER(%s.%s) LIKE ? OR LOWER(%s.%s) LIKE ? OR EXTRACT(DAY FROM %s.created_at) = ?",
			tableAlias, titleCol, tableAlias, statementCol, tableAlias, nameCol, tableAlias)
		
		args := []interface{}{
			"%"+searchText+"%", // title
			"%"+searchText+"%", // statement
			"%"+searchText+"%", // name
			*dateResult.Day,     // day of month
		}
		
		return whereClause, args
	} else {
		// Text search only
		whereClause := fmt.Sprintf("LOWER(%s.%s) LIKE ? OR LOWER(%s.%s) LIKE ? OR LOWER(%s.%s) LIKE ?",
			tableAlias, titleCol, tableAlias, statementCol, tableAlias, nameCol)
		
		args := []interface{}{
			"%"+searchText+"%", // title
			"%"+searchText+"%", // statement
			"%"+searchText+"%", // name
		}
		
		return whereClause, args
	}
}
