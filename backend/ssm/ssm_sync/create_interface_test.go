package ssmsync

// Compile-time checks to ensure creators implement CreateKeySSM.
var (
	_ CreateKeySSM = (*CreateAES128SSM)(nil)
	_ CreateKeySSM = (*CreateAES256SSM)(nil)
	_ CreateKeySSM = (*CreateDes3SSM)(nil)
	_ CreateKeySSM = (*CreateDesSSM)(nil)
)
