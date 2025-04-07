package utils

func init() {
	// Ensure OssObjectManager implements ObjectManager interface
	var _ ObjectManager = (*OssObjectManager)(nil)
}
