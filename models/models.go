package models

import (
	"time"
)

// User represents a user in the system (student, center admin, or super admin)
type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Don't expose password hash in JSON
	FullName     string    `json:"full_name" db:"full_name"`
	PhoneNumber  string    `json:"phone_number" db:"phone_number"`
	Avatar       *string   `json:"avatar,omitempty" db:"avatar"`
	Role         string    `json:"role" db:"role"`           // 'super_admin', 'center_admin', 'student'
	CenterID     *string   `json:"center_id" db:"center_id"` // ID of assigned center (for center admins & activated students)
	Status       string    `json:"status" db:"status"`       // 'pending', 'active', 'inactive' (for students)
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Email    string  `json:"email" validate:"required,email"`
	Password string  `json:"password" validate:"required,min=8"`
	FullName string  `json:"full_name" validate:"required"`
	Role     string  `json:"role" validate:"required,oneof=super_admin center_admin student"`
	CenterID *string `json:"center_id,omitempty"` // Optional center ID for center admins
}

// LoginRequest represents the login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	User      User   `json:"user"`
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// StudentRegisterRequest represents the student registration request
type StudentRegisterRequest struct {
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// StudentLoginRequest represents the student login request
type StudentLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UpdateStudentProfileRequest represents the request to update a student's profile
type UpdateStudentProfileRequest struct {
	FullName *string `json:"full_name,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Avatar   *string `json:"avatar,omitempty"`
}

// StudentProfileResponse represents the student profile returned to the client
type StudentProfileResponse struct {
	ID        string  `json:"id"`
	FullName  string  `json:"full_name"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	Avatar    *string `json:"avatar,omitempty"`
	Status    string  `json:"status"`
	IsActive  bool    `json:"is_active"`
	CreatedAt string  `json:"created_at"`
}

// StudentSearchResult represents the admin's view of a student during activation search
type StudentSearchResult struct {
	ID          string  `json:"id"`
	FullName    string  `json:"full_name"`
	Email       string  `json:"email"`
	Phone       string  `json:"phone"`
	CenterID    *string `json:"center_id"`
	Status      string  `json:"status"`
	IsActive    bool    `json:"is_active"`
	CreatedAt   string  `json:"created_at"`
	BelongsToUs bool    `json:"belongs_to_us"` // true if center_id matches admin's center
}

// EducationalCenter represents an educational center
type EducationalCenter struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Description   string    `json:"description" db:"description"`
	AdminUserID   *string   `json:"admin_user_id" db:"admin_user_id"`               // ID of center admin
	AdminFullName *string   `json:"admin_full_name,omitempty" db:"admin_full_name"` // Full name of center admin
	IsActive      bool      `json:"is_active" db:"is_active"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// CreateEducationalCenterRequest represents the request to create an educational center
type CreateEducationalCenterRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	AdminUserID string `json:"admin_user_id"`
}

// StudentCenterEnrollment represents a student's enrollment in a center
type StudentCenterEnrollment struct {
	ID             string    `json:"id" db:"id"`
	StudentID      string    `json:"student_id" db:"student_id"`
	CenterID       string    `json:"center_id" db:"center_id"`
	EnrollmentDate time.Time `json:"enrollment_date" db:"enrollment_date"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// Exam represents an exam created by a center admin
type Exam struct {
	ID              string    `json:"id" db:"id"`
	Title           string    `json:"title" db:"title"`
	Description     string    `json:"description" db:"description"` // Subject (e.g., ریاضی, فزیک)
	Class           string    `json:"class" db:"class"`             // Grade/class (10, 11, 12)
	CenterID        string    `json:"center_id" db:"center_id"`
	CreatorID       string    `json:"creator_id" db:"creator_id"` // Center admin who created
	StartTime       time.Time `json:"start_time" db:"start_time"`
	EndTime         time.Time `json:"end_time" db:"end_time"`
	DurationMinutes int       `json:"duration_minutes" db:"duration_minutes"`
	IsPublished     bool      `json:"is_published" db:"is_published"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	Status          string    `json:"status" db:"status"` // "pending" or "published"
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// CreateExamRequest represents the request to create an exam
type CreateExamRequest struct {
	Title           string `json:"title"`                                    // Optional exam title
	Description     string `json:"description" validate:"required"`          // Subject (required, e.g., ریاضی, فزیک)
	Class           string `json:"class" validate:"required,oneof=10 11 12"` // Grade/class (required)
	CenterID        string `json:"center_id" validate:"required"`
	Date            string `json:"date" validate:"required"`       // "2026-04-10"
	StartTime       string `json:"start_time" validate:"required"` // "10:00"
	DurationMinutes int    `json:"duration_minutes" validate:"required,min=1,max=360"`
	Status          string `json:"status"` // "pending" or "published"
}

// Question represents an MCQ question in an exam
type Question struct {
	ID            string    `json:"id" db:"id"`
	ExamID        string    `json:"exam_id" db:"exam_id"`
	QuestionText  string    `json:"question_text" db:"question_text"`
	QuestionOrder int       `json:"question_order" db:"question_order"`
	Score         float64   `json:"score" db:"score"` // Points for this question
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// CreateQuestionRequest represents the request to create a question
type CreateQuestionRequest struct {
	ExamID        string                        `json:"exam_id" validate:"required"`
	QuestionText  string                        `json:"question_text" validate:"required"`
	QuestionOrder int                           `json:"question_order" validate:"required"`
	Score         float64                       `json:"score" validate:"required,min=0"` // Points for this question
	Options       []CreateQuestionOptionRequest `json:"options"`
}

// QuestionOption represents an option for an MCQ question
type QuestionOption struct {
	ID           string    `json:"id" db:"id"`
	QuestionID   string    `json:"question_id" db:"question_id"`
	OptionText   string    `json:"option_text" db:"option_text"`
	OptionLetter string    `json:"option_letter" db:"option_letter"` // 'A', 'B', 'C', 'D'
	IsCorrect    bool      `json:"is_correct" db:"is_correct"`
	OptionOrder  int       `json:"option_order" db:"option_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// CreateQuestionOptionRequest represents the request to create a question option
type CreateQuestionOptionRequest struct {
	QuestionID   string `json:"question_id" validate:"required"`
	OptionText   string `json:"option_text" validate:"required"`
	OptionLetter string `json:"option_letter" validate:"required,oneof=A B C D"`
	IsCorrect    bool   `json:"is_correct"`
	OptionOrder  int    `json:"option_order" validate:"required"`
}

// ExamAttempt represents a student's attempt at an exam
type ExamAttempt struct {
	ID               string     `json:"id" db:"id"`
	ExamID           string     `json:"exam_id" db:"exam_id"`
	StudentID        string     `json:"student_id" db:"student_id"`
	StartTime        time.Time  `json:"start_time" db:"start_time"`
	SubmitTime       *time.Time `json:"submit_time" db:"submit_time"`
	TimeTakenSeconds *int       `json:"time_taken_seconds" db:"time_taken_seconds"`
	Score            *float64   `json:"score" db:"score"`
	TotalQuestions   int        `json:"total_questions" db:"total_questions"`
	CorrectAnswers   *int       `json:"correct_answers" db:"correct_answers"`
	IsSubmitted      bool       `json:"is_submitted" db:"is_submitted"`
	IsTimedOut       bool       `json:"is_timed_out" db:"is_timed_out"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// StudentAnswer represents a student's answer to a question
type StudentAnswer struct {
	ID               string     `json:"id" db:"id"`
	AttemptID        string     `json:"attempt_id" db:"attempt_id"`
	QuestionID       string     `json:"question_id" db:"question_id"`
	SelectedOptionID *string    `json:"selected_option_id" db:"selected_option_id"`
	IsCorrect        *bool      `json:"is_correct" db:"is_correct"`
	AnsweredAt       *time.Time `json:"answered_at" db:"answered_at"`
	TimeSpentSeconds *int       `json:"time_spent_seconds" db:"time_spent_seconds"`
}

// SubmitAnswerRequest represents the request to submit an answer
type SubmitAnswerRequest struct {
	SelectedOptionID string `json:"selected_option_id" validate:"required"`
}

// OTPCode represents an OTP code for authentication
type OTPCode struct {
	ID          string    `json:"id" db:"id"`
	PhoneNumber string    `json:"phone_number" db:"phone_number"`
	OTPCode     string    `json:"-" db:"otp_code"`      // Don't expose OTP in JSON
	Purpose     string    `json:"purpose" db:"purpose"` // 'register', 'login', 'reset_password'
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
	IsUsed      bool      `json:"is_used" db:"is_used"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// RefreshToken represents a refresh token
type RefreshToken struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	TokenHash string    `json:"-" db:"token_hash"` // Don't expose token hash
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	IsRevoked bool      `json:"is_revoked" db:"is_revoked"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// SystemSetting represents a system setting
type SystemSetting struct {
	ID           string    `json:"id" db:"id"`
	SettingKey   string    `json:"setting_key" db:"setting_key"`
	SettingValue string    `json:"setting_value" db:"setting_value"`
	Description  string    `json:"description" db:"description"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// QuestionWithFlatOptions represents an exam question with options flattened to A,B,C,D format
// Used for exam engine where questions have simple option structure (NEW schema)
type QuestionWithFlatOptions struct {
	ID            string    `json:"id" db:"id"`
	ExamID        string    `json:"exam_id" db:"exam_id"`
	Text          string    `json:"text" db:"text"`
	OptionA       string    `json:"option_a" db:"option_a"`
	OptionB       string    `json:"option_b" db:"option_b"`
	OptionC       string    `json:"option_c" db:"option_c"`
	OptionD       string    `json:"option_d" db:"option_d"`
	CorrectOption string    `json:"correct_option" db:"correct_option"` // A, B, C, or D
	Score         float64   `json:"score" db:"score"`                   // Points for this question
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// StudentAnswerSimple represents a student's answer (simplified version for exam engine)
// Used with NEW database schema (answers table)
type StudentAnswerSimple struct {
	ID             string    `json:"id" db:"id"`
	StudentID      string    `json:"student_id" db:"student_id"`
	ExamID         string    `json:"exam_id" db:"exam_id"`
	QuestionID     string    `json:"question_id" db:"question_id"`
	SelectedOption string    `json:"selected_option" db:"selected_option"` // A, B, C, or D
	IsCorrect      bool      `json:"is_correct" db:"is_correct"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// QuestionWithOptions represents a question with formatted options for API response
type QuestionWithOptions struct {
	ID      string            `json:"id"`
	Text    string            `json:"text"`
	Options map[string]string `json:"options"` // {"A": "Option A text", "B": "Option B text", ...}
}

// QuestionReview represents a single question's review data
type QuestionReview struct {
	QuestionID     string  `json:"question_id"`
	Question       string  `json:"question"`
	SelectedOption *string `json:"selected_option"`
	CorrectOption  string  `json:"correct_option"`
	IsCorrect      bool    `json:"is_correct"`
}

// ExamResultResponse represents the response after submitting an exam
type ExamResultResponse struct {
	Score               float64          `json:"score"`
	CorrectAnswers      int              `json:"correct_answers"`
	TotalQuestions      int              `json:"total_questions"`
	WrongAnswers        int              `json:"wrong_answers"`
	UnansweredQuestions int              `json:"unanswered_questions"`
	Percentage          float64          `json:"percentage"`
	Passed              bool             `json:"passed"`
	TimeSpent           int              `json:"time_spent"`
	QuestionsReview     []QuestionReview `json:"questions_review"`
}

// SaveAnswerRequest represents the request to save a student's answer
type SaveAnswerRequest struct {
	ExamID         string `json:"exam_id" validate:"required"`
	QuestionID     string `json:"question_id" validate:"required"`
	SelectedOption string `json:"selected_option" validate:"required,oneof=A B C D"`
}

// SubmitExamRequest represents the request to submit an exam
type SubmitExamRequest struct {
	ExamID    string `json:"exam_id" validate:"required"`
	TimeSpent int    `json:"time_spent_seconds"`
}

// ExamResult represents the result of a student's exam attempt (database model)
type ExamResult struct {
	ID                  string    `json:"id" db:"id"`
	StudentID           string    `json:"student_id" db:"student_id"`
	ExamID              string    `json:"exam_id" db:"exam_id"`
	Score               float64   `json:"score" db:"score"`
	TotalQuestions      int       `json:"total_questions" db:"total_questions"`
	CorrectAnswers      int       `json:"correct_answers" db:"correct_answers"`
	WrongAnswers        int       `json:"wrong_answers" db:"wrong_answers"`
	UnansweredQuestions int       `json:"unanswered_questions" db:"unanswered_questions"`
	TimeSpentSeconds    int       `json:"time_spent_seconds" db:"time_spent_seconds"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// QuestionBankEntry represents a question in the question bank
type QuestionBankEntry struct {
	ID            string    `json:"id" db:"id"`
	CenterID      string    `json:"center_id" db:"center_id"`
	Subject       string    `json:"subject" db:"subject"`
	QuestionText  string    `json:"question_text" db:"question_text"`
	OptionA       string    `json:"option_a" db:"option_a"`
	OptionB       string    `json:"option_b" db:"option_b"`
	OptionC       string    `json:"option_c" db:"option_c"`
	OptionD       string    `json:"option_d" db:"option_d"`
	CorrectOption *string   `json:"correct_option" db:"correct_option"`
	CreatedBy     string    `json:"created_by" db:"created_by"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// QuestionBankUploadRequest represents the request to upload a PDF for question extraction
type QuestionBankUploadRequest struct {
	Subject string `json:"subject" validate:"required"`
}

// GeminiExtractedQuestion represents a question extracted by Gemini AI
type GeminiExtractedQuestion struct {
	Question      string            `json:"question"`                 // Legacy field
	QuestionText  string            `json:"question_text"`            // New field
	Options       map[string]string `json:"options"`                  // Legacy field: {"A": "...", "B": "...", ...}
	OptionA       string            `json:"option_a"`                 // New field
	OptionB       string            `json:"option_b"`                 // New field
	OptionC       string            `json:"option_c"`                 // New field
	OptionD       string            `json:"option_d"`                 // New field
	CorrectOption string            `json:"correct_option,omitempty"` // Set manually by admin after review (not populated by AI)
	Invalid       bool              `json:"invalid,omitempty"`        // Mark if data is unclear
}

// SaveQuestionBankRequest represents the request to save extracted questions to the question bank
type SaveQuestionBankRequest struct {
	Subject   string                    `json:"subject" validate:"required"`
	Questions []GeminiExtractedQuestion `json:"questions" validate:"required,min=1"`
}

// UpdateQuestionBankRequest represents the request to update a question bank entry
type UpdateQuestionBankRequest struct {
	Subject       *string `json:"subject,omitempty"`
	QuestionText  *string `json:"question_text,omitempty"`
	OptionA       *string `json:"option_a,omitempty"`
	OptionB       *string `json:"option_b,omitempty"`
	OptionC       *string `json:"option_c,omitempty"`
	OptionD       *string `json:"option_d,omitempty"`
	CorrectOption *string `json:"correct_option,omitempty"`
}
