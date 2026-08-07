package dto

type DeclarationPDFInfo struct {
	Statement string `json:"statement"`
	Response  string `json:"response"`
}

// DeclarationFormPDFInfo represents a group of declarations under a form
type DeclarationFormPDFInfo struct {
	FormID      uint                  `json:"form_id"`
	FormLabel   string                `json:"form_label"`
	Declarations []DeclarationPDFInfo `json:"declarations"`
}

type ConsentFormPDFRequest struct {
	NHI                    string               `json:"nhi"`
	FirstName              string               `json:"first_name"`
	LastName               string               `json:"last_name"`
	DOB                    string               `json:"dob"`
	Gender                 string               `json:"gender"`
	Ethnicity              string               `json:"ethnicity"`
	PreferredCommunication string               `json:"preferred_communication"`
	TrackDate              string               `json:"track_date"`
	ProjectName            string               `json:"project_name"`
	QuestionnaireName      string               `json:"questionnaire_name"`
	CompletedAt            string               `json:"completed_at"`
	MilestoneName          string               `json:"milestone_name"`
	DeclarationForms       []DeclarationFormPDFInfo `json:"declaration_forms"`
	PatientSignature       string               `json:"patient_signature"`
	PatientSignDate        string               `json:"patient_sign_date"`
	ResearcherName         string               `json:"researcher_name"`
	ResearcherSignature    string               `json:"researcher_signature"`
	ResearcherSignDate     string               `json:"researcher_sign_date"`
}