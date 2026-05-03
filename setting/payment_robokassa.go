package setting

var (
	RobokassaEnabled       bool
	RobokassaMerchantLogin string
	RobokassaPassword1     string
	RobokassaPassword2     string
	RobokassaTestPassword1 string
	RobokassaTestPassword2 string
	RobokassaSandbox       bool
	RobokassaCurrency      string  = "RUB"
	RobokassaUnitPrice     float64 = 1.0
	RobokassaMinTopUp      int     = 1
	RobokassaSignatureAlgo string  = "md5"
	RobokassaResultUrl     string
	RobokassaSuccessUrl    string
	RobokassaFailUrl       string
)
