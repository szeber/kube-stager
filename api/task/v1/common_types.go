package v1

type EnvironmentConfig struct {
	// Name of the service for this database. Empty for the main app
	//+optional
	ServiceName string `json:"serviceName,omitempty"`
	//+kubebuilder:validation:MinLength=1
	// Name of the site this database is associated with
	SiteName string `json:"siteName"`
	//+kubebuilder:validation:MinLength=1
	// Name of the environment used
	Environment string `json:"environment"`
}

func (r EnvironmentConfig) GetSiteName() string {
	return r.SiteName
}

func (r EnvironmentConfig) GetServiceName() string {
	return r.ServiceName
}

func (r EnvironmentConfig) GetEnvironment() string {
	return r.Environment
}

type TaskStatus struct {
	// The state of the task. Pending/Failed/Complete
	State TaskState `json:"state"`
}

type TaskState string

const (
	Pending  TaskState = "Pending"
	Failed   TaskState = "Failed"
	Complete TaskState = "Complete"
)
