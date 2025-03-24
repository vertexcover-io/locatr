package tracing

type Option interface {
	applyOption(*config) *config
}

type optionFunc func(*config) *config

func (optFnc optionFunc) applyOption(cfg *config) *config {
	return optFnc(cfg)
}

type config struct {
	endpoint string
	svcName  string
	insecure bool
}

func WithEndpoint(endpoint string) Option {
	return optionFunc(func(c *config) *config {
		c.endpoint = endpoint
		return c
	})
}

func WithSVCName(svcName string) Option {
	return optionFunc(func(c *config) *config {
		c.svcName = svcName
		return c
	})
}

func WithInsecure(insecure bool) Option {
	return optionFunc(func(c *config) *config {
		c.insecure = insecure
		return c
	})
}

func WithDefaults() Option {
	return optionFunc(func(c *config) *config {
		c.endpoint = DEFAULT_ENDPOINT
		c.svcName = DEFAULT_SVC_NAME
		c.insecure = DEFAULT_INSECURE
		return c
	})
}
