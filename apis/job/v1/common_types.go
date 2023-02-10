package v1

type JobState string

const (
	Pending  JobState = "Pending"
	Running  JobState = "Running"
	Failed   JobState = "Failed"
	Complete JobState = "Complete"
)

func (r JobState) IsFinal() bool {
	if r == Failed || r == Complete {
		return true
	}

	return false
}
