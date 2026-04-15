//go:build !windows

package platform

func AcquireSingleton(name string) (release func(), err error) {
	return func() {}, nil
}
