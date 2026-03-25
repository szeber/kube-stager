package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	jobv1 "github.com/szeber/kube-stager/apis/job/v1"
	sitev1 "github.com/szeber/kube-stager/apis/site/v1"
	taskv1 "github.com/szeber/kube-stager/apis/task/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	stagingSitesDesc = prometheus.NewDesc(
		"kube_stager_staging_sites",
		"Current number of StagingSite resources by state and enabled status.",
		[]string{"namespace", "state", "enabled"},
		nil,
	)
	databasesDesc = prometheus.NewDesc(
		"kube_stager_databases",
		"Current number of database task resources by type and state.",
		[]string{"namespace", "type", "state"},
		nil,
	)
	jobsDesc = prometheus.NewDesc(
		"kube_stager_jobs",
		"Current number of job resources by kind and state.",
		[]string{"namespace", "kind", "state"},
		nil,
	)
)

// ResourceCollector implements prometheus.Collector to provide resource inventory
// gauges by reading from the informer cache on each scrape.
type ResourceCollector struct {
	reader client.Reader
}

// NewResourceCollector creates a new ResourceCollector.
func NewResourceCollector(reader client.Reader) *ResourceCollector {
	return &ResourceCollector{reader: reader}
}

func (c *ResourceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- stagingSitesDesc
	ch <- databasesDesc
	ch <- jobsDesc
}

func (c *ResourceCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c.collectStagingSites(ctx, ch)
	c.collectDatabases(ctx, ch)
	c.collectJobs(ctx, ch)
}

func (c *ResourceCollector) collectStagingSites(ctx context.Context, ch chan<- prometheus.Metric) {
	var list sitev1.StagingSiteList
	if err := c.reader.List(ctx, &list); err != nil {
		ch <- prometheus.NewInvalidMetric(stagingSitesDesc, err)
		return
	}

	type siteKey struct {
		namespace string
		state     string
		enabled   string
	}

	counts := map[siteKey]float64{}
	for _, site := range list.Items {
		state := string(site.Status.State)
		if state == "" {
			state = string(sitev1.StatePending)
		}
		enabled := fmt.Sprintf("%t", site.Status.Enabled)
		counts[siteKey{site.Namespace, state, enabled}]++
	}

	for key, count := range counts {
		m, err := prometheus.NewConstMetric(stagingSitesDesc, prometheus.GaugeValue, count, key.namespace, key.state, key.enabled)
		if err != nil {
			ch <- prometheus.NewInvalidMetric(stagingSitesDesc, err)
			return
		}
		ch <- m
	}
}

func (c *ResourceCollector) collectDatabases(ctx context.Context, ch chan<- prometheus.Metric) {
	type dbEntry struct {
		namespace string
		dbType    string
		state     string
	}

	counts := map[dbEntry]float64{}

	var mysqlList taskv1.MysqlDatabaseList
	if err := c.reader.List(ctx, &mysqlList); err != nil {
		ch <- prometheus.NewInvalidMetric(databasesDesc, err)
		return
	}
	for _, db := range mysqlList.Items {
		state := string(db.Status.State)
		if state == "" {
			state = string(taskv1.Pending)
		}
		counts[dbEntry{db.Namespace, "mysql", state}]++
	}

	var mongoList taskv1.MongoDatabaseList
	if err := c.reader.List(ctx, &mongoList); err != nil {
		ch <- prometheus.NewInvalidMetric(databasesDesc, err)
		return
	}
	for _, db := range mongoList.Items {
		state := string(db.Status.State)
		if state == "" {
			state = string(taskv1.Pending)
		}
		counts[dbEntry{db.Namespace, "mongo", state}]++
	}

	var redisList taskv1.RedisDatabaseList
	if err := c.reader.List(ctx, &redisList); err != nil {
		ch <- prometheus.NewInvalidMetric(databasesDesc, err)
		return
	}
	for _, db := range redisList.Items {
		state := string(db.Status.State)
		if state == "" {
			state = string(taskv1.Pending)
		}
		counts[dbEntry{db.Namespace, "redis", state}]++
	}

	for entry, count := range counts {
		m, err := prometheus.NewConstMetric(databasesDesc, prometheus.GaugeValue, count, entry.namespace, entry.dbType, entry.state)
		if err != nil {
			ch <- prometheus.NewInvalidMetric(databasesDesc, err)
			return
		}
		ch <- m
	}
}

func (c *ResourceCollector) collectJobs(ctx context.Context, ch chan<- prometheus.Metric) {
	type jobEntry struct {
		namespace string
		kind      string
		state     string
	}

	counts := map[jobEntry]float64{}

	var initList jobv1.DbInitJobList
	if err := c.reader.List(ctx, &initList); err != nil {
		ch <- prometheus.NewInvalidMetric(jobsDesc, err)
		return
	}
	for _, j := range initList.Items {
		state := string(j.Status.State)
		if state == "" {
			state = string(jobv1.Pending)
		}
		counts[jobEntry{j.Namespace, "dbinit", state}]++
	}

	var migrationList jobv1.DbMigrationJobList
	if err := c.reader.List(ctx, &migrationList); err != nil {
		ch <- prometheus.NewInvalidMetric(jobsDesc, err)
		return
	}
	for _, j := range migrationList.Items {
		state := string(j.Status.State)
		if state == "" {
			state = string(jobv1.Pending)
		}
		counts[jobEntry{j.Namespace, "dbmigration", state}]++
	}

	var backupList jobv1.BackupList
	if err := c.reader.List(ctx, &backupList); err != nil {
		ch <- prometheus.NewInvalidMetric(jobsDesc, err)
		return
	}
	for _, j := range backupList.Items {
		state := string(j.Status.State)
		if state == "" {
			state = string(jobv1.Pending)
		}
		counts[jobEntry{j.Namespace, "backup", state}]++
	}

	for entry, count := range counts {
		m, err := prometheus.NewConstMetric(jobsDesc, prometheus.GaugeValue, count, entry.namespace, entry.kind, entry.state)
		if err != nil {
			ch <- prometheus.NewInvalidMetric(jobsDesc, err)
			return
		}
		ch <- m
	}
}
