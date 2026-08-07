// utils/constantsgo
package utils

const (
	// Route Names

	// Public
	RouteWelcome               = "/"
	RouteLogin                 = "/login"
	RoutePatientRegister       = "/patient-register"
	RouteForgetPassword        = "/user/forgot-password"
	RouteEncryptProductDetails = "/encrypt-product"
	RouteVerifyProductDetails  = "/verify-product"
	RouteProductPaymentDetails = "/order-payment"
	RoutePaymentSuccessPaypal  = "/payment/status"
	RouteKitRegistration       = "/kit-registration"
	RouteFetchQuantityDiscount = "/quantity-discounts"
	RouteVerifyBarcode         = "/verify-barcode"
	RouteKitRegister           = "/kit-register"
	RouteEncryptData           = "/encrypt"

	// Private
	RouteLogout         = "/logout"
	RouteResetPassword  = "/user/reset-password"
	RouteUpdateUser     = "/user"
	RouteGetUserProfile = "/user/profile"

	RouteGetAdminPatientList       = "/patient"
	RouteCreateAdminPatient        = "/patient"
	RouteUpdateAdminPatientStatus  = "/patient/:id/change-status"
	RouteUpdateAdminPatientDetails = "/patient/:id/change-details"

	RouteGetAdminUserList           = "/staff"
	RouteCreateAdminUser            = "/staff"
	RouteUpdateAdminUserProfile     = "/staff/:id/change-details"
	RouteUpdateAdminUserPassword    = "/staff/:id/change-password"
	RouteUpdateAdminUserStatus      = "/staff/:id/change-status"
	RouteDeleteAdminUser            = "/staff/:id"
	RouteKitInfo                    = "/kits"
	RouteKitInfoID                  = "/kits/:id"
	RouteQuantitySummary            = "/kits/quantity-summary"
	RouteCustomer                   = "/customers"
	RouteviewCustomerWithOrders     = "/customers/:id/orders"
	RouteOrder                      = "/orders"
	RouteAssignKit                  = "/orders/:id/assign-kit"
	RouteChangeOrderStatus          = "/orders/:id/change-status"
	RouteOrderCount                 = "/orders/count"
	RouteNotification               = "/notifications"
	RouteLatestNotification         = "/notifications/latest"
	RouteMarkNotificationAsRead     = "/notifications/:id/mark-read"
	RouteMarkAllNotificationsAsRead = "/notifications/mark-all-read"
	RouteDeleteNotification         = "/notifications/:id"
	RouteFetchedKitRegistrationList = "/patients-list"
	RoutePatientUpdateStatus        = "/patients/:id/change-status"
	RoutePatientUploadReport        = "/patients/:id/upload-report"
	RouteFetchedBarcodeDetails      = "/barcodes/:id/details"
	RouteFetchedLabsDetails         = "/labs"

	// Payment
	PaymentStatusPending   = "Pending"
	PaymentStatusCompleted = "Completed"
	PaymentStatusFailed    = "Failed"
	PaymentStatusCancelled = "Cancelled"

	OrderStatusPending    = "Pending"
	OrderStatusProcessing = "Processing"
	OrderStatusCancelled  = "Cancelled"
	OrderStatusFailed     = "Failed"

	// API Messages
	MsgWelcome                                 = "Welcome to the project!"
	MsgDatabaseConnected                       = "Database connection established successfully."
	MsgDatabaseMigrated                        = "Database migration completed successfully."
	MsgServerStarted                           = "Server is running on port %s."
	MsgInternalServerError                     = "An internal server error has occurred."
	MsgEndpointNotFound                        = "The requested endpoint could not be found."
	MsgUserNotFound                            = "User not found in the current context."
	MsgInvalidRequest                          = "The request is invalid."
	MsgUserNotAuthenticated                    = "User is not authenticated."
	MsgInvalidJSONData                         = "Invalid JSON data provided."
	MsgInvalidFormData                         = "Invalid form data provided."
	MsgUnsupportedContentType                  = "Unsupported content type. Only JSON and form-urlencoded data are allowed."
	MsgOnlyEmailPasswordAllowed                = "Only email and password fields are permitted."
	MsgOnlyAllowedFieldsAllowed                = "Only specific fields are permitted."
	MsgAccessDeniedForOtherThanAdminSuperAdmin = "You are not authorized to view this page."

	// Seeding Messages
	MsgRolesSeededSuccessfully     = "Roles seeded successfully."
	MsgRoleCreated                 = "Role '%s' created successfully."
	MsgRoleAlreadyExists           = "Role '%s' already exists."
	MsgFailedToCreateRole          = "Failed to create role '%s': %v."
	MsgFailedToCheckRole           = "Error checking role '%s': %v."
	MsgSeedingCompleted            = "Seeding process completed successfully."
	MsgFailedToCreateDiscount      = "Failed to create discount for quantity %d with percentage %.2f: %v"
	MsgDiscountCreated             = "Discount created successfully for quantity %d with percentage %.2f"
	MsgFailedToCheckDiscount       = "Failed to check existing discount for quantity %d: %v"
	MsgDiscountAlreadyExists       = "Discount already exists for quantity %d"
	MsgDiscountsSeededSuccessfully = "Quantity discounts seeded successfully"

	// User Seeding Messages
	MsgUsersSeededSuccessfully = "Users seeded successfully."
	MsgUserCreated             = "User '%s' created successfully."
	MsgUserAlreadyExists       = "User '%s' already exists."
	MsgFailedToCreateUser      = "Failed to create user '%s': %v."
	MsgFailedToCheckUser       = "Error checking user '%s': %v."

	// Middlewares Messages
	MsgAuthHeaderMissing            = "Authorization header missing"
	MsgInvalidAuthHeaderFormat      = "Invalid authorization header format"
	MsgInvalidTokenClaims           = "Invalid token claims."
	MsgUserDoesNotExist             = "User does not exist."
	MsgUserAccountInactiveOrDeleted = "User account is inactive or deleted."
	MsgUnauthorizedUser             = "Unauthorized user."

	// Notification message
	MsgFailedToSendNotification           = "Failed to send Notification"
	MsgFailedToSendNotificationUser       = "Failed to send notification to the user"
	MsgFailedToNotifySuperAdmins          = "Failed to notify super admins about password update"
	MsgPasswordUpdatedSuccessfully        = "Password updated successfully"
	MsgInvalidNotificationType            = "Invalid notification type parameter"
	MsgInvalidIsReadParameter             = "Invalid is_read parameter"
	MsgNotificationsFetchedSuccess        = "Notifications fetched successfully"
	MsgNotificationIDRequired             = "Notification ID is required"
	MsgInvalidNotificationID              = "Invalid Notification ID"
	MsgIsReadRequired                     = "`is_read` is required and must be true or false"
	MsgNotificationNotFound               = "Notification not found"
	MsgNotificationAlreadySet             = "Notification is already set to the requested status"
	MsgFailedToUpdateNotification         = "Failed to update notification"
	MsgNotificationUpdatedSuccessfully    = "Notification updated successfully"
	MsgFailedToCountUnreadNotifications   = "Failed to count unread notifications"
	MsgNoUnreadNotificationsToMarkAsRead  = "No unread notifications to mark as read"
	MsgFailedToMarkAllNotificationsAsRead = "Failed to mark all notifications as read"
	MsgAllUnreadNotificationsMarkedAsRead = "All unread notifications marked as read successfully"
	MsgInvalidDateFormat                  = "Invalid date format. Expected format is YYYY-MM-DD."
	MsgInvalidStartDate                   = "Invalid start date."
	MsgInvalidEndDate                     = "Invalid end date."
	MsgInvalidDateRange                   = "Invalid date range."
	MsgDateRangeRequired                  = "Start date and end date are required."
	MsgInvalidStartDateFormat             = "Invalid start_date format. Expected format is YYYY-MM-DD."
	MsgInvalidEndDateFormat               = "Invalid end_date format. Expected format is YYYY-MM-DD."
	MsgNoNotificationsFoundForDateRange   = "No notifications found for the given date range."
	MsgFailedToDeleteNotifications        = "Failed to delete notifications."
	MsgNotificationsDeletedSuccess        = "Notifications deleted successfully."
	NotificationDeletedMessage            = "Notification deleted successfully"
	NotificationNotFoundError             = "Notification not found or already deleted"
	ValidationError                       = "User ID and Notification ID are required"
	InternalServerError                   = "Failed to delete notification"
	MsgNotificationMarkedAsRead           = "Notification marked as read."

	// Login Messages
	MsgLoginSuccess               = "Login Successful."
	MsgInvalidCredentials         = "Invalid email or password."
	MsgInvalidEmailFormat         = "The provided email format is invalid."
	MsgInvalidPasswordFormat      = "Password must be 8-20 characters long and include at least one uppercase letter, one lowercase letter, one number, and one special character."
	MsgMissingEmailOrPassword     = "Both email and password are required."
	MsgTokenCreationFailed        = "Failed to create the JWT token."
	MsgAuthorizationHeaderMissing = "Authorization header is missing."
	MsgInvalidAuthorizationHeader = "The format of the authorization header is invalid."
	MsgInvalidOrExpiredToken      = "The token is either invalid or expired."
	MsgUserInactive               = "The account is currently inactive."
	MsgTokenSaveFailed            = "Failed to save the token. Please try again."
	MsgLogoutSuccess              = "Successfully logged out."

	// Forget Password Messages
	MsgMissingEmail       = "Email is required."
	MsgEmailNotExist      = "The provided email does not exist."
	MsgFailedSentEmail    = "Failed to send email. Please try again later."
	MsgForgetSuccessfully = "A new password has been sent successfully. Please check your email."

	// Reset Password Messages
	MsgPasswordSame              = "The new password must differ from the old password."
	MsgInvalidOldPassword        = "The provided old password is invalid."
	MsgFailedToHashNewPassword   = "Failed to hash the new password."
	MsgFailedToUpdateNewPassword = "Failed to update the new password."
	MsgResetPasswordSuccessfully = "The password has been reset successfully."
	MsgOldNewPasswordRequired    = "Both old and new passwords are required."

	// Update User Messages
	MsgAtleastPassSomeData     = "First name must be provided."
	MsgInvalidRoleId           = "The provided role ID is invalid."
	MsgUserUpdateSuccessfully  = "User information updated successfully."
	MsgUserFetchedSuccessfully = "User profile fetched successfully."

	// Fetch Admin User Details Messages
	MsgAccessDeniedForOtherUser    = "You are not authorized to view this page."
	MsgStatusInvalid               = "The status filter is invalid. Allowed values: all, active, inactive."
	MsgRoleInvalid                 = "The role is invalid. Allowed values: patient, coordinator."
	MsgFailedToCountRecords        = "Failed to count the records."
	MsgFailedToFetchRecords        = "Failed to fetch the records."
	MsgUserListFetchedSuccessfully = "Users fetched successfully."
	MsgAdminRoleNotFound           = "Admin role not found."

	// Create Admin User Messages
	MsgFirstNameValidation     = "First name must be alphabetic and between 3 to 20 characters long."
	MsgLastNameValidation      = "Last name must be alphabetic and between 3 to 20 characters long (if provided)."
	MsgEmailValidation         = "Email format is invalid or exceeds the maximum length."
	MsgEmailAlreadyInUse       = "The email is already in use."
	MsgFailedToSecurePassword  = "Failed to secure the password."
	MsgFailedCreateUser        = "Failed to create the user."
	MsgFailedToSendEmail       = "Failed to send email."
	MsgUserCreatedSuccessfully = "User created successfully."
	MsgFirstNameEmailRequired  = "Both first name and email are required."

	// Update Admin User Messages
	MsgPasswordValidation          = "Password must be between 8 to 20 characters and include one uppercase letter, one lowercase letter, one number, and one special character."
	MsgFailedToUpdateAdminUser     = "Failed to update the user."
	MsgAdminUserUpdateSuccessfully = "User updated successfully."

	// Delete Admin User Messages
	MsgUserAlreadyDeleted     = "User not found or has already been deleted."
	MsgFailedToDeleteUser     = "Failed to delete the user."
	MsgUserDeleteSuccessfully = "User deleted successfully."

	// Kit Related Messages
	MsgKitCreatedSuccessfully      = "Kit added successfully."
	MsgInvalidKitType              = "Invalid kit type. Must be either 'blood' or 'saliva'."
	MsgValidationSupplierName      = "Supplier name should be alphabetic and between 3 to 50 characters."
	MsgValidationContactNumber     = "Contact number should be numeric and between 10 to 15 digits."
	MsgValidationAddress           = "Address must be between 5 to 100 characters."
	MsgAllFieldsOfkitRequired      = "All fields are required."
	MsgKitsListFetchedSuccessfully = "Kits list fetched successfully."
	MsgTotalKitsQunatitiesFetched  = "Total kit quantities fetched successfully"

	MsgKitUpdatedSuccessfully              = "Kit details updated successfully."
	MsgKitDeletedSuccessfully              = "Kit deleted successfully."
	MsgKitNotFound                         = "Kit not found."
	MsgKitAlreadyDeleted                   = "Kit not found or already deleted."
	MsgInvalidQueryParameters              = "Invalid query parameters."
	MsgInvalidPageParameter                = "Invalid 'page' parameter."
	MsgInvalidPerPageParameter             = "Invalid 'per_page' parameter."
	MsgInvalidSortParameter                = "Invalid 'sort' parameter."
	MsgInvalidSortColumnParameter          = "Invalid 'sort_column' parameter."
	MsgQuantityCannotBeEmpty               = "Quantity cannot be empty."
	MsgInvalidQuantityFormat               = "Invalid quantity format."
	MsgInvalidQuantityType                 = "Invalid quantity type."
	MsgQuantityCannotBeNegative            = "Quantity cannot be negative."
	MsgQuantityExceedsMaxValue             = "Quantity exceeds maximum allowed value."
	MsgTypeCannotBeEmptyIfProvided         = "Type cannot be empty if provided."
	MsgSupplierNameCannotBeEmptyIfProvided = "Supplier name cannot be empty if provided."
	MsgInvalidKitID                        = "Invalid kit ID."

	// Payment Related Message
	MsgFailedToStartTransaction       = "Failed to start transaction"
	MsgFailedToProcessCustomer        = "Failed to process customer: %s"
	MsgFailedToInitializePayment      = "Failed to initialize payment: %s"
	MsgFailedToCommitTransaction      = "Failed to commit transaction"
	MsgOrderCreatedSuccessfully       = "Order created successfully"
	MsgInvalidFirstName               = "Invalid first name: must be 3-50 characters long and contain only letters"
	MsgInvalidEmail                   = "Invalid email."
	MsgInvalidLastName                = "Invalid last name: must be 3-50 characters long and contain only letters"
	MsgInvalidPhoneNumber             = "Invalid phone number: must be 10-15 digits"
	MsgInvalidCountryName             = "Invalid country name: must be 3-50 characters"
	MsgInvalidStreetAddress           = "Invalid street address: must be 5-255 characters"
	MsgInvalidTownCity                = "Invalid town/city: must be 5-100 characters"
	MsgInvalidRegion                  = "Invalid region: must be 3-100 characters"
	MsgInvalidPostcode                = "Invalid postcode: must be 3-20 characters"
	MsgInvalidShippingCountryName     = "Invalid shipping country name: must be 3-50 characters"
	MsgInvalidShippingAddress         = "Invalid shipping address: must be 5-255 characters"
	MsgInvalidShippingTownCity        = "Invalid shipping town/city: must be 5-100 characters"
	MsgInvalidShippingRegion          = "Invalid shipping region: must be 3-100 characters"
	MsgInvalidShippingPostcode        = "Invalid shipping postcode: must be 3-20 characters"
	MsgInvalidProductName             = "Invalid product name: must be 3-100 characters and contain only letters, numbers, and basic punctuation"
	MsgProductDescriptionTooLong      = "Product description too long: must not exceed 1000 characters"
	MsgInvalidProductImage            = "Invalid product image: must be a valid base64 encoded string"
	MsgInvalidProductPrice            = "Invalid product price: must be a positive number with a maximum of 2 decimal places"
	MsgInvalidProductGstPrice         = "Invalid product GST price: must be a positive number with a maximum of 2 decimal places"
	MsgInvalidQuantity                = "Invalid quantity: must be a positive integer"
	MsgInvalidPriceFormat             = "Invalid price format"
	MsgMissingPaymentInformation      = "Missing payment information"
	MsgFailedToStartTransactionAgain  = "Failed to start transaction"
	MsgPaymentVerificationFailed      = "Payment verification failed: %s"
	MsgFailedToCompletePaymentProcess = "Failed to complete payment process"
	MsgPaymentCompletedSuccessfully   = "Payment completed successfully"
	MsgPaymentNotFound                = "Payment not found"
	MsgFailedToUpdatePayment          = "Failed to update payment"
	MsgOrderNotFound                  = "Order not found"
	MsgInvalidImageFormat             = "Invalid image format."

	MsgErrorEncodingProductDetails          = "Error encoding product details"
	MsgErrorEncryptingProductDetails        = "Error encrypting product details"
	MsgProductDetailsEncryptedSuccessfully  = "Product details encrypted successfully"
	MsgMissingOrEmptyEncryptedDataParameter = "Missing or empty encrypted data parameter"
	MsgFailedToDecryptData                  = "Failed to decrypt data"
	MsgInvalidProductDataFormat             = "Invalid product data format"
	MsgProductNameIsRequired                = "Product name is required"
	MsgProductPriceMustBeGreaterThanZero    = "Product price must be greater than zero"
	MsgProductGstPriceMustBeGreaterThanZero = "Product  GST price must be greater than zero"
	MsgProductVerifiedSuccessfully          = "Product verified successfully"
	MsgInvalidCustomerType                  = "Invalid type provided. Must be 'customer' or 'clinic'"
	MsgClinicIDRequired                     = "Clinic ID is required for type 'clinic'"
	MsgClinicIDInvalidLength                = "Clinic ID cannot exceed 100 characters"
	MsgCustomerNotBuyMoreThenOne            = "Customer can not buy more then one kit."
	MsgClinicInvalidQuantity                = "Clinic can order qunatity should be more then equal to 25."

	// customer related message
	MsgCustomerFetchedSuccessfully = "Customer details fetched successfully"
	MsgInvalidCustomerId           = "Invalid customer ID"
	MsgCustomerNotFound            = "Customer not found"

	// order related message
	MsgInvalidPaymentStatus          = "Invalid payment status"
	MsgInvalidOrderStatus            = "Invalid order status"
	MsgOrdersFetchedSuccessfully     = "Orders details fetched successfully"
	MsgInvalidOrderStatusReq         = "Invalid status. Allowed values are: Pending, Processing, Shipped, Delivered, Cancelled, Dispatched"
	MsgOrderIdRequired               = "Order ID is required"
	MsgOrderAlreadySetToSameValue    = "Order status is already set to the same value"
	MsgBarcodeNotAssigned            = "QRCode/Barcode not assigned to the order"
	MsgOrderStatusUpdateSuccessfully = "Order status updated successfully"
	MsgOrderCountFetchedSuccessfully = "Order counts fetched successfully"

	// Assigned Kit related message
	MsgInvalidBarcodeLength        = "QRCode/Barcode number must be 14 characters long"
	MsgInvalidBarcodeFormat        = "QRCode/Barcode number must be alphanumeric"
	MsgBarcodeAlreadyAssigned      = "QRCode/Barcode is already assigned"
	MsgInvalidOrderId              = "Invalid order ID format"
	MsgBarcodeAssignedSuccessfully = "QRCode/Barcode successfully assigned"
	MsgPaymentnotCompleted         = "Payment not completed for the order"

	// Qunatity and Discount API related message
	MsgFailedToFetchQuantityAndDiscount  = "Failed to fetch quantity discounts"
	MsgSuccessToFetchQuantityAndDiscount = "Fetched quantity and discount list successfully"

	MsgBarcodeNumberRequired         = "QRCode/Barcode number is required."
	MsgPatientDataRequired           = "Patient data is required."
	MsgInvalidPatientDataFormat      = "Invalid patient data format"
	MsgPatientDataStoredSuccessfully = "Your kit has been successfully registered."
	MsgInvalidGender                 = "Invalid gender. Male, Female or Other accepted."
	MsgInvalidAge                    = "Invalid age. Age should atleast 18"
	MsgBarcodeOrOrderNotFound        = "Barcodes not found."
	MsgKitRegistrationExists         = "Patient data already exist for this barcode."
	MsgInvalidKitStatusParmas        = "Invalid status : Not Received, Received, Reject"

	/****************** E Consent Routes ******************/
	// Video Routes
	RouteVideoUpload = "/videos/upload"
	RouteVideoList   = "/videos"
	// RouteVideoGet    = "/videos/:id"
	RouteVideoDelete = "/videos/:id"
	RouteVideoUpdate = "/videos/:id"

	// Admin Dashboard
	RouteAdminDashboardStats        = "/admin/dashboard/stats"
	RouteAdminDashboardDropDownData = "/admin/e-consent/dropdown-data"

	// Declaration Routes
	RouteDeclarationCreate = "/declarations"
	RouteDeclarationList   = "/declarations"
	RouteDeclarationUpdate = "/declarations/:id"
	RouteDeclarationDelete = "/declarations/:id"

	// Research Category Routes
	RouteResearchCategoryCreate = "/research-categories"
	RouteResearchCategoryList   = "/research-categories"

	// Research Routes
	RouteResearchCreate  = "/research"
	RouteResearchList    = "/research"
	RouteResearchDelete  = "/research/:id"
	RouteResearchPublish = "/research/:id/publish"

	// E-Consent Routes
	RouteEConsentCreate  = ""
	RouteEConsentEdit    = "/e-consent/:id"
	RouteEConsentPublish = "/e-consent/:id/publish"

	// Patient Management
	RoutePatientManagementList          = "/patient-managements"
	RoutePatientEConsent                = "/patient-econsent"
	RoutePatientConsentPDF              = "/patient-management/:patient_econsent_id/pdf"
	RoutePatientEConsentRequestResign   = "/patient-econsent/request-resign"
	RoutePatientEConsentAdminSign       = "/patient-econsent/:id/admin-sign"
	RoutePatientMyEConsents             = "/patient/my-econsents"
	RoutePatientEConsentResign          = "/patient-econsent/:id/resign"
	RoutePatientEConsentFormDetails     = "/patient/econsent-form/:id"
	RoutePatientEConsentAssignedList    = "/patient/assigned-econsents"
	RoutePatientEConsentManageWithdrawl = "/patient-management/econsent/:id/manage-withdrawn"

	// Patient Routes
	RoutePatientEconsent              = "/patient-econsent"
	RoutePatientEconsentRequestReSign = "/patient-econsent/request-resign"
	RoutePatientEconsentID            = "/patient/econsent-form/:id"
	RoutePatientEconsentAssigned      = "/patient/assigned-econsents"
	RoutePatientEconsentWithdrawal    = "/patient/econsent/:id/withdraw"
	RoutePatientGenomicResults        = "/patient/genomic-results"

	RouteAdminPatientEconsentID = "/admin/econsent-form/:id/:patientId"

	//verification Patient Routes

	// Genomic Routes
	RouteGenomicList           = "/genomic/consented-patients"
	RouteGenomicResultUpload   = "/genomic-result"
	RouteGenomicResults        = "/genomics"
	RouteGenomicResultEdit     = "/genomics/:id"
	RouteGenomicResultDelete   = "/genomics/:id"
	RouteGenomicResultPublish  = "/genomics/:id/publish"
	RouteGenomicResultServe    = "/genomic-result/view/:filename"
	RouteGenomicResultDownload = "/genomic-result/download/:filename"

	// Static Routes
	RouteStaticVideos      = "/videos"
	RouteStaticResearchDoc = "/research_documents"
	RouteStaticGenomicDoc  = ""

	//Consent form
	RouteCreateSession                   = "/consent-session"
	RouteFormDetails                     = "/form"
	RouteResearchDocument                = "/research-document"
	RoutePatientEconsentSendVerification = "/send-verification"
	RoutePatientEconsentVerifyCode       = "/verify-code"
	RouteSubmitConsentForm               = "/submit"
)

// Video Constants
const (
	MaxVideoSize     = 100 * 1024 * 1024 // 100MB
	AllowedVideoMime = "video/mp4,video/quicktime,video/x-msvideo,video/x-matroska"
	VideoUploadPath  = "public/videos"
)
