package ssmsync

// Compile-time checks to ensure creators implement CreateKeySSM.
var _ CreateKeySSM = (*CreateAES128SSM)(nil)
var _ CreateKeySSM = (*CreateAES256SSM)(nil)
var _ CreateKeySSM = (*CreateDes3SSM)(nil)
var _ CreateKeySSM = (*CreateDesSSM)(nil)
