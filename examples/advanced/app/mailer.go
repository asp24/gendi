package app

type Mailer interface {
	Send(message string) error
}

type MailerBasic struct {
	Host string
}

func NewMailerBasic(host string) Mailer {
	return &MailerBasic{Host: host}
}

func (m *MailerBasic) Send(message string) error {
	return nil
}

type MailerRetryDecorator struct {
	inner   Mailer
	Retries int
}

func NewMailerRetryDecorator(inner Mailer, retries int) *MailerRetryDecorator {
	return &MailerRetryDecorator{inner: inner, Retries: retries}
}

func (m *MailerRetryDecorator) Send(message string) error {
	for i := 0; i <= m.Retries; i++ {
		if err := m.inner.Send(message); err == nil || i == m.Retries {
			return err
		}
	}
	return nil
}

type MailerPrefixDecorator struct {
	inner  Mailer
	Prefix string
}

func NewMailerPrefixDecorator(inner Mailer, prefix string) *MailerPrefixDecorator {
	return &MailerPrefixDecorator{inner: inner, Prefix: prefix}
}

func (m *MailerPrefixDecorator) Send(message string) error {
	return m.inner.Send(m.Prefix + message)
}
