package upstream

import (
	"net/url"

	"github.com/payfazz/iso8585-utility-lib/upstream/spec"
)

// Builder .
type Builder struct {
	inner *Upstream
}

// NewBuilder .
func NewBuilder() *Builder {
	return &Builder{inner: &Upstream{}}
}

// WithTarget .
func (b *Builder) WithTarget(target string) *Builder {
	b.inner.target = target
	return b
}

// WithSpec .
func (b *Builder) WithSpec(spec spec.Spec) *Builder {
	b.inner.spec = spec
	return b
}

// WithLogger .
func (b *Builder) WithLogger(info, err func(string)) *Builder {
	b.inner.logger.Info = info
	b.inner.logger.Err = err
	return b
}

// WithProxy .
func (b *Builder) WithProxy(proxy *url.URL, proxyCASum string) *Builder {
	b.inner.proxy.endpoint = proxy
	b.inner.proxy.caSum = proxyCASum
	return b
}
