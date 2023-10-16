package importer

type ImportType string

const (
	TYPE_IMPORT_EXPORT_DATA ImportType = "importExportData"
	TYPE_STAGING_SITE       ImportType = "stagingSite"
	TYPE_MYSQL_DATABASE     ImportType = "mysqlDatabase"
	TYPE_MONGO_DATABASE     ImportType = "mongoDatabase"
	TYPE_REDIS_DATABASE     ImportType = "redisDatabase"
	TYPE_DB_INIT_JOB        ImportType = "dbInitJob"
	TYPE_DB_MIGRATION_JOB   ImportType = "dbMigrationJob"
	TYPE_BACKUP_JOB         ImportType = "backupJob"
)
