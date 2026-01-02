package app

type Notifier interface {
	Notify() error
}

type EmailNotifier struct {
	Mailer Mailer
}

func NewEmailNotifier(mailer Mailer) *EmailNotifier {
	return &EmailNotifier{Mailer: mailer}
}

func (n *EmailNotifier) Notify() error { return nil }

type SMSNotifier struct{}

func NewSMSNotifier() SMSNotifier { return SMSNotifier{} }

func (n SMSNotifier) Notify() error { return nil }

type AggregateNotifier struct {
	notifiers []Notifier
}

func NewAggregateNotifier(notifiers []Notifier) *AggregateNotifier {
	return &AggregateNotifier{notifiers: notifiers}
}

func (n *AggregateNotifier) Notify() error {
	for _, n := range n.notifiers {
		if err := n.Notify(); err != nil {
			return err
		}
	}

	return nil
}
