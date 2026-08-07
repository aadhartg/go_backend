package utils

import (
	"bytes"
	"os"
	"theransticslabs/m/dto"

	"github.com/jung-kurt/gofpdf"
)

// GenerateConsentPDF creates a 3-page PDF from the consent form data, mimicking the provided layout
func GenerateConsentPDF(data dto.ConsentFormPDFRequest) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	margin := 10.0
	// pageWidth, _ := pdf.GetPageSize()
	// usableWidth := pageWidth - 2*margin

	// Table column widths (shared by all tables for alignment)
	labelWidth := 95.0
	valueWidth := 95.0
	cellHeight := 8.0

	// --- Page 1: Header, Common Info, Responses ---
	pdf.AddPage()
	pdf.SetMargins(margin, margin, margin)

	// Header: Logo (optional) and Title with color 
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(117, 172, 113) // green
	pdf.CellFormat(0, 10, "Theranostics", "0", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.SetTextColor(117, 172, 113) // Greenish
	pdf.CellFormat(0, 8, "Health New Zealand", "0", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0) // Reset to black
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 12, "Questionnaire Response PDF", "0", 1, "C", false, 0, "")
	pdf.Ln(2)

	// Common Information Table (aligned)
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(0, 8, "Common Information", "0", 1, "L", false, 0, "")
	pdf.Ln(1)
	pdf.SetFont("Arial", "", 11)
	// Helper to ensure no empty field
	getOrNA := func(s string) string {
		if s == "" {
			return "N/A"
		}
		return s
	}
	infoRows := [][]string{
		{"NHI", getOrNA(data.NHI)},
		{"First name", getOrNA(data.FirstName)},
		{"Last name", getOrNA(data.LastName)},
		{"Date of birth", getOrNA(data.DOB)},
		{"Gender", getOrNA(data.Gender)},
		// {"Ethnicity", getOrNA(data.Ethnicity)},
		{"Preferred Mode of Communication", "Phone"},
		{"Track date", getOrNA(data.TrackDate)},
		{"Project name", getOrNA(data.ProjectName)},
		{"Questionnaire name", "Test"},
		{"Completed at", getOrNA(data.CompletedAt)},
		{"Milestone name", "Baseline"},
	}
	for _, row := range infoRows {
		x := pdf.GetX()
		y := pdf.GetY()

		labelLines := pdf.SplitLines([]byte(row[0]), labelWidth)
		valueLines := pdf.SplitLines([]byte(row[1]), valueWidth)
		labelHeight := float64(len(labelLines)) * cellHeight
		valueHeight := float64(len(valueLines)) * cellHeight
		rowHeight := labelHeight
		if valueHeight > labelHeight {
			rowHeight = valueHeight
		}

		// Draw label cell
		pdf.SetXY(x, y)
		pdf.SetFont("Arial", "B", 11)
		pdf.MultiCell(labelWidth, rowHeight/float64(len(labelLines)), row[0], "1", "L", false)

		// Draw value cell
		pdf.SetXY(x+labelWidth, y)
		pdf.SetFont("Arial", "", 11)
		pdf.MultiCell(valueWidth, rowHeight/float64(len(valueLines)), row[1], "1", "L", false)

		// Move to next row
		pdf.SetY(y + rowHeight)
	}
	pdf.Ln(6) // More space between tables

	// Responses Section (Consent Form Table, aligned)
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(0, 8, "Responses", "0", 1, "L", false, 0, "")
	pdf.Ln(1)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Consent Form", "0", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	// Dummy description row (spans both columns)
	desc := "Clinical Genomics Implementation and Research Programme (CLINGEN)\nPlease check that your details below are correct before you proceed, inform the nurse if any corrections are needed"
	pdf.MultiCell(labelWidth+valueWidth, 6, desc, "1", "L", false)
	pdf.Ln(1)
	pdf.SetFont("Arial", "", 11)
	respRows := [][]string{
		{"NHI Number:", data.NHI},
		{"First Name:", data.FirstName},
		{"Last Name:", data.LastName},
		{"Gender:", data.Gender},
		{"Date of Birth:", data.DOB},
	}
	for _, row := range respRows {
		x := pdf.GetX()
		y := pdf.GetY()

		labelLines := pdf.SplitLines([]byte(row[0]), labelWidth)
		valueLines := pdf.SplitLines([]byte(row[1]), valueWidth)
		labelHeight := float64(len(labelLines)) * cellHeight
		valueHeight := float64(len(valueLines)) * cellHeight
		rowHeight := labelHeight
		if valueHeight > labelHeight {
			rowHeight = valueHeight
		}

		pdf.SetXY(x, y)
		pdf.SetFont("Arial", "B", 11)
		pdf.MultiCell(labelWidth, rowHeight/float64(len(labelLines)), row[0], "1", "L", false)

		pdf.SetXY(x+labelWidth, y)
		pdf.SetFont("Arial", "", 11)
		pdf.MultiCell(valueWidth, rowHeight/float64(len(valueLines)), row[1], "1", "L", false)

		pdf.SetY(y + rowHeight)
	}

	// --- Page 2: Declarations and Optional Consent ---
	pdf.AddPage()
	pdf.SetMargins(margin, margin, margin)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "DECLARATIONS", "0", 1, "L", false, 0, "")
	pdf.Ln(1)
	pdf.SetFont("Arial", "", 11)

	for _, form := range data.DeclarationForms {
		// Form label as section header
		pdf.SetFont("Arial", "B", 11)
		pdf.CellFormat(0, 8, form.FormLabel, "0", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 11)
		for _, decl := range form.Declarations {
			x := pdf.GetX()
			y := pdf.GetY()

			labelLines := pdf.SplitLines([]byte(decl.Statement), labelWidth)
			valueLines := pdf.SplitLines([]byte(decl.Response), valueWidth)
			labelHeight := float64(len(labelLines)) * cellHeight
			valueHeight := float64(len(valueLines)) * cellHeight
			rowHeight := labelHeight
			if valueHeight > labelHeight {
				rowHeight = valueHeight
			}

			// Draw statement cell
			pdf.SetXY(x, y)
			pdf.MultiCell(labelWidth, rowHeight/float64(len(labelLines)), decl.Statement, "1", "L", false)

			// Draw answer cell
			pdf.SetXY(x+labelWidth, y)
			pdf.SetFont("Arial", "", 11)
			pdf.MultiCell(valueWidth, rowHeight/float64(len(valueLines)), decl.Response, "1", "C", false)

			// Move to next row
			pdf.SetY(y + rowHeight)
		}
		pdf.Ln(2)
	}

	// Optional Consent Section
	// pdf.SetFont("Arial", "B", 12)
	// pdf.CellFormat(labelWidth+valueWidth, cellHeight, "OPTIONAL CONSENT FOR FUTURE RESEARCH AND CONTACT", "1", 1, "C", false, 0, "")
	// pdf.SetFont("Arial", "", 11)
	optionalRows := [][]string{
		// {"I consent to future unspecified research on my stored sample, provided it aligns with rare cardiac disease screening or personalized medicine and has ethical approval.", boolToYesNo(data.ConsentFutureResearch)},
		// {"I consent to being contacted in the future regarding additional research or clinical initiatives related to this program.", boolToYesNo(data.ConsentFutureContact)},
	}
	for _, row := range optionalRows {
		x := pdf.GetX()
		y := pdf.GetY()

		labelLines := pdf.SplitLines([]byte(row[0]), labelWidth)
		valueLines := pdf.SplitLines([]byte(row[1]), valueWidth)
		labelHeight := float64(len(labelLines)) * cellHeight
		valueHeight := float64(len(valueLines)) * cellHeight
		rowHeight := labelHeight
		if valueHeight > labelHeight {
			rowHeight = valueHeight
		}

		pdf.SetXY(x, y)
		pdf.MultiCell(labelWidth, rowHeight/float64(len(labelLines)), row[0], "1", "L", false)

		pdf.SetXY(x+labelWidth, y)
		pdf.SetFont("Arial", "", 11)
		pdf.MultiCell(valueWidth, rowHeight/float64(len(valueLines)), row[1], "1", "C", false)

		pdf.SetY(y + rowHeight)
	}
	pdf.Ln(4)

	// Declaration by participant
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(labelWidth+valueWidth, cellHeight, "Declaration by participant:", "1", 1, "L", false, 0, "")
	// Add consent statement
	pdf.SetFont("Arial", "", 11)
	pdf.MultiCell(labelWidth+valueWidth, 7, "I hereby consent to take part in this Clinical Genomics Implementation and Research Programme (CLINGEN)", "1", "L", false)
	// Participant details table
	pdf.CellFormat(labelWidth, cellHeight, "Participant's Name:", "1", 0, "L", false, 0, "")
	pdf.CellFormat(valueWidth, cellHeight, data.FirstName+" "+data.LastName, "1", 1, "L", false, 0, "")
	pdf.CellFormat(labelWidth, cellHeight, "Signature:", "1", 0, "L", false, 0, "")
	// Embed patient signature image if exists
	if data.PatientSignature != "" {
		imgPath := data.PatientSignature
		if imgPath[0] == '/' {
			imgPath = "public" + imgPath // convert to local path
		}
		if _, err := os.Stat(imgPath); err == nil {
			pdf.Image(imgPath, pdf.GetX(), pdf.GetY(), 30, 12, false, "", 0, "")
			pdf.Ln(12)
		} else {
			pdf.CellFormat(valueWidth, cellHeight, "[Signature not found]", "1", 1, "L", false, 0, "")
		}
	} else {
		pdf.CellFormat(valueWidth, cellHeight, "[No signature]", "1", 1, "L", false, 0, "")
	}
	pdf.CellFormat(labelWidth, cellHeight, "Date:", "1", 0, "L", false, 0, "")
	pdf.CellFormat(valueWidth, cellHeight, data.PatientSignDate, "1", 1, "L", false, 0, "")

	// Add space before remaining sample selection
	pdf.Ln(2)

	// Remaining sample selection (fixed for wrapping and alignment)
	// pdf.SetFont("Arial", "B", 11)
	// label := "PLEASE SELECT ONE OF THE FOLLOWING FOR ANY REMAINING SAMPLE:"
	// value := data.RemainingSampleOption
	// labelLines := pdf.SplitLines([]byte(label), labelWidth)
	// labelHeight := float64(len(labelLines)) * cellHeight
	// pdf.SetFont("Arial", "", 11)
	// valueLines := pdf.SplitLines([]byte(value), valueWidth)
	// valueHeight := float64(len(valueLines)) * cellHeight
	// rowHeight := labelHeight
	// if valueHeight > labelHeight {
	// 	rowHeight = valueHeight
	// }
	// x := pdf.GetX()
	// y := pdf.GetY()
	// pdf.SetFont("Arial", "B", 11)
	// pdf.SetXY(x, y)
	// pdf.MultiCell(labelWidth, rowHeight/float64(len(labelLines)), label, "1", "L", false)
	// pdf.SetFont("Arial", "", 11)
	// pdf.SetXY(x+labelWidth, y)
	// pdf.MultiCell(valueWidth, rowHeight/float64(len(valueLines)), value, "1", "L", false)
	// pdf.SetY(y + rowHeight)

	// --- Page 3: Research Team Declaration ---
	// pdf.AddPage()
	pdf.SetMargins(margin, margin, margin)
	pdf.SetFont("Arial", "B", 11)
	// Table header spanning both columns
	pdf.CellFormat(labelWidth+valueWidth, cellHeight, "Declaration by member of research team:", "1", 1, "L", false, 0, "")
	// Add explanation statement
	pdf.SetFont("Arial", "", 11)
	pdf.MultiCell(labelWidth+valueWidth, 7, "I have given a verbal explanation of the Clinical Genomics Implementation and Research Programme (CLINGEN) to the participant, answered the participant's questions, and believe the participant understands the program and has given informed consent to participate.", "1", "L", false)
	// Researcher details table
	pdf.CellFormat(labelWidth, cellHeight, "Researcher's Name:", "1", 0, "L", false, 0, "")
	pdf.CellFormat(valueWidth, cellHeight, data.ResearcherName, "1", 1, "L", false, 0, "")
	pdf.CellFormat(labelWidth, cellHeight, "Signature:", "1", 0, "L", false, 0, "")
	// Embed nurse/researcher signature image if exists
	if data.ResearcherSignature != "" {
		imgPath := data.ResearcherSignature
		if imgPath[0] == '/' {
			imgPath = "public" + imgPath // convert to local path
		}
		if _, err := os.Stat(imgPath); err == nil {
			pdf.Image(imgPath, pdf.GetX(), pdf.GetY(), 30, 12, false, "", 0, "")
			pdf.Ln(12)
		} else {
			pdf.CellFormat(valueWidth, cellHeight, "[Signature not found]", "1", 1, "L", false, 0, "")
		}
	} else {
		pdf.CellFormat(valueWidth, cellHeight, "[No signature]", "1", 1, "L", false, 0, "")
	}
	pdf.CellFormat(labelWidth, cellHeight, "Date:", "1", 0, "L", false, 0, "")
	pdf.CellFormat(valueWidth, cellHeight, data.ResearcherSignDate, "1", 1, "L", false, 0, "")
	pdf.Ln(4)

	// Info box for "Are you sure you do not wish to consent?"
	pdf.SetFont("Arial", "B", 10)
	pdf.MultiCell(0, 7, "Are you sure that you do not wish to consent?\nYou have selected 'No' for at least one of the consent questions. If you have any questions, please speak to the clinician available at the clinic, or contact the team via:\n- Phone: (as above)\n- Email: (Research Nurse Coordinator's email)\n- Click 'BACK' to review your selections OR click 'NEXT' to end the form", "1", "L", false)

	// Output PDF to bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Helper for YES/NO
func boolToYesNo(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
} 