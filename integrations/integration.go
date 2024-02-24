package integrations

type Integration interface {
	Play() error
	Pause() error
	SetValue(string, int) error
	LongPress(string) error
	Setup() error
}
