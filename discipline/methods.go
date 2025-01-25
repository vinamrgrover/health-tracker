package discipline

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/redis/go-redis/v9"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func InitHealth() (*Health, error) {
	db, err := sql.Open("sqlite3", "/tmp/wellness.db")

	if err != nil {
		return nil, err
	}

	return &Health{
		db,
	}, nil
}

func Init() (*Queue, error) {
	ctx := context.Background()

	err := godotenv.Load()

	if err != nil {
		return nil, err
	}

	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisUser := os.Getenv("REDIS_USER")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	addr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: redisUser,
		Password: redisPassword,
		DB:       0,
	})

	if err := rdb.Ping(ctx); err.Err() != nil {
		rdb.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %v", err.Err())
	}

	return &Queue{
		rdb,
		"",
		context.Background(),
	}, nil
}

func (q *Queue) LogTrigger(triggerType string, intensity int, compulsion bool, notes string) error {
	q.Key = "health-triggers"
	now := time.Now().UTC()
	timestamp := now.Format(time.RFC3339)

	t := Trigger{
		Id:         uuid.New().String(),
		Timestamp:  timestamp,
		Type:       triggerType,
		Intensity:  intensity,
		Compulsion: compulsion,
		Notes:      notes,
	}

	jsonData, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("failed to marshal trigger data: %w", err)
	}

	// Handle LPush error
	if err := q.Client.LPush(q.Ctx, q.Key, jsonData).Err(); err != nil {
		return fmt.Errorf("failed to push to Redis: %w", err)
	}

	return nil
}

func (q *Queue) LogAction(actionType string, relief bool, duration int, notes string) error {

	q.Key = "health-actions"
	now := time.Now().UTC()
	timestamp := now.Format(time.RFC3339)

	a := Action{
		Id:        uuid.New().String(),
		Timestamp: timestamp,
		Type:      actionType,
		Relief:    relief,
		Duration:  duration,
		Notes:     notes,
	}

	jsonData, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshal trigger data: %w", err)
	}

	// Handle LPush error
	if err := q.Client.LPush(q.Ctx, q.Key, jsonData).Err(); err != nil {
		return fmt.Errorf("failed to push to Redis: %w", err)
	}

	return nil
}

func (q *Queue) LogOverall(cleanStreak int, gymSession bool, codingHours int, readingHours int, moodScore int) error {

	q.Key = "health-overall"
	now := time.Now().UTC()
	timestamp := now.Format(time.RFC3339)

	o := Overall{
		Id:           uuid.New().String(),
		Timestamp:    timestamp,
		CleanStreak:  cleanStreak,
		GymSession:   gymSession,
		CodingHours:  codingHours,
		ReadingHours: readingHours,
		MoodScore:    moodScore,
	}

	jsonData, err := json.Marshal(o)
	if err != nil {
		return fmt.Errorf("failed to marshal trigger data: %w", err)
	}

	// Handle LPush error
	if err := q.Client.LPush(q.Ctx, q.Key, jsonData).Err(); err != nil {
		return fmt.Errorf("failed to push to Redis: %w", err)
	}

	return nil
}

func (q *Queue) PopAndInsert(name string) (*string, error) {
	val, err := q.Client.LPop(q.Ctx, name).Result()

	if err != nil {
		return nil, err
	}
	return &val, err
}

func InitDatabase(local bool) (*Database, error) {

	log.Println("removing file - /tmp/health.sqlite")
	log.Println("removing file - /tmp/out.sqlite")

	if err := os.Remove("/tmp/health.sqlite"); err != nil {
		log.Println("unable to remove /tmp/health.sqlite")
	}

	if err := os.Remove("/tmp/out.sqlite"); err != nil {
		log.Println("unable to remove /tmp/out.sqlite")
	}

	if local {

		db, err := sql.Open("sqlite3", "./tmp/health.sqlite")

		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)

		return &Database{
			db,
			ctx,
			cancel,
		}, nil
	}

	if err := ImportSqlite(); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", "/tmp/health.sqlite")

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)

	return &Database{db, ctx, cancel}, nil

}

func SinkType[T Overall | Action | Trigger](q *Queue, name string) (T, error) {
	var res T

	val, err := q.Client.LPop(q.Ctx, name).Result()

	if err != nil {
		return res, fmt.Errorf("error while unmarshalling - %v\n", err)
	}

	if err := json.Unmarshal([]byte(val), &res); err != nil {
		return res, err
	}

	return res, nil
}

func GetSink[T Overall | Action | Trigger](q *Queue, name string) []T {
	var arr []T
	for {
		res, err := SinkType[T](q, name)

		if err != nil {
			break
		}

		arr = append(arr, res)
	}
	return arr
}

func InsertRecords[T Overall | Action | Trigger](db *sql.DB, records []T) error {
	// Skip if no records
	if len(records) == 0 {
		return nil
	}

	// Determine table name and query based on type
	var query string
	switch any(records[0]).(type) {
	case Overall:
		query = `INSERT INTO overall 
            (id, timestamp, clean_streak, gym_session, coding_hours, reading_hours, mood_score) 
            VALUES (?, ?, ?, ?, ?, ?, ?)`

	case Trigger:
		query = `INSERT INTO trigger 
            (id, timestamp, type, intensity, compulsion, notes) 
            VALUES (?, ?, ?, ?, ?, ?)`

	case Action:
		query = `INSERT INTO action 
            (id, timestamp, type, relief, duration, notes) 
            VALUES (?, ?, ?, ?, ?, ?)`

	default:
		return fmt.Errorf("unsupported type for insertion")
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Prepare statement
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert records
	for _, record := range records {
		var err error
		switch v := any(record).(type) {
		case Overall:
			_, err = stmt.Exec(
				v.Id, v.Timestamp, v.CleanStreak, v.GymSession,
				v.CodingHours, v.ReadingHours, v.MoodScore,
			)

		case Trigger:
			_, err = stmt.Exec(
				v.Id, v.Timestamp, v.Type, v.Intensity,
				v.Compulsion, v.Notes,
			)

		case Action:
			_, err = stmt.Exec(
				v.Id, v.Timestamp, v.Type, v.Relief,
				v.Duration, v.Notes,
			)
		}

		if err != nil {
			return fmt.Errorf("failed to insert record: %v", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func ProcessAndStore(db *sql.DB, q *Queue) error {
	// Process Overall records
	overalls := GetSink[Overall](q, "health-overall")
	log.Printf("found %d records in health-overall queue", len(overalls))
	if err := InsertRecords(db, overalls); err != nil {
		return fmt.Errorf("failed to insert overall records: %w", err)
	}

	// Process Trigger records
	triggers := GetSink[Trigger](q, "health-triggers")
	log.Printf("found %d records in health-triggers queue", len(triggers))
	if err := InsertRecords(db, triggers); err != nil {
		return fmt.Errorf("failed to insert trigger records: %w", err)
	}

	// Process Action records
	actions := GetSink[Action](q, "health-actions")
	log.Printf("found %d records in health-actions queue", len(actions))
	if err := InsertRecords(db, actions); err != nil {
		return fmt.Errorf("failed to insert action records: %w", err)
	}

	return nil
}

func TableExists(ctx context.Context, db *sql.DB, tableName string) (bool, error) {
	query := `
        SELECT COUNT(name) 
        FROM sqlite_master 
        WHERE type='table' AND name=?
    `

	var count int
	err := db.QueryRowContext(ctx, query, tableName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking table existence: %w", err)
	}

	return count > 0, nil
}

func (db *Database) checkExistence() bool {
	// checks if the tables are present
	for _, name := range []string{"trigger", "action", "overall"} {
		exists, _ := TableExists(db.Ctx, db.Db, name)
		if !exists {
			return false
		}
	}
	return true
}

func ImportSqlite() error {

	bucketName := "health-tracking-utility"
	keyName := "db/health.sqlite"

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)

	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)

	out, err := os.Create("/tmp/health.sqlite")

	if err != nil {
		return err
	}

	defer out.Close()

	res, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: &bucketName, Key: &keyName})

	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") {
			_, err := os.Create("/tmp/health.sqlite")
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	defer res.Body.Close()

	_, err = io.Copy(out, res.Body)

	if err != nil {
		return err
	}

	return nil
}

func ExportSqlite() error {
	bucketName := "health-tracking-utility"
	keyName := "db/health.sqlite"

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile("vinamrgrover"))

	if err != nil {
		return err
	}

	client := s3.NewFromConfig(cfg)

	file, err := os.Open("/tmp/out.sqlite")

	if err != nil {
		return err
	}

	defer file.Close()

	_, err = client.PutObject(ctx, &s3.PutObjectInput{Bucket: &bucketName, Key: &keyName, Body: file})

	if err != nil {
		return err
	}

	return nil
}

func CreateTables(db *Database) error {
	file, err := os.Open("tables.sql")

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	contents, err := io.ReadAll(file)

	if err != nil {
		return err
	}

	if _, err := db.Db.Exec(string(contents)); err != nil {
		return err
	}

	return nil
}

func saveDBToFile(db *sql.DB, filePath string) error {
	// Backup the current in-memory database to a file
	_, err := os.Stat(filePath)

	if os.IsExist(err) {
		if err := os.Remove("/tmp/out.sqlite"); err != nil {
			return err
		}

		if err := os.Remove("/tmp/health.sqlite"); err != nil {
			return err
		}
	}
	_, err = db.Exec("VACUUM INTO ?", filePath)
	return err
}

func RefreshSink() error {
	db, err := InitDatabase(false)

	if err != nil {
		return err
	}

	q, err := Init()

	if err != nil {
		return err
	}

	log.Println("checking if tables exists")
	if db.checkExistence() {
		log.Println("yes, tables exists")
		err := ProcessAndStore(db.Db, q)

		if err != nil {
			return err
		}

		log.Println("saving locally")
		err = saveDBToFile(db.Db, "/tmp/out.sqlite")

		if err != nil {
			return err
		}

		log.Println("exporting to s3")
		if err := ExportSqlite(); err != nil {
			return err
		}
		return nil
	}

	log.Println("tables do not exist")
	log.Println("creating tables")

	if err := CreateTables(db); err != nil {
		log.Printf("unable to create tables - %v\n", err)
		return err
	}

	log.Println("writing from queue")
	err = ProcessAndStore(db.Db, q)

	if err != nil {
		return err
	}

	log.Println("saving locally")
	err = saveDBToFile(db.Db, "/tmp/out.sqlite")

	if err != nil {
		return err
	}

	log.Println("exporting to s3")
	if err := ExportSqlite(); err != nil {
		return err
	}

	return nil
}
