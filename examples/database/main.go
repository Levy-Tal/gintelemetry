package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Levy-Tal/gintelemetry"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func main() {
	ctx := context.Background()

	// Initialize telemetry
	tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
		ServiceName: "database-example",
		Endpoint:    "localhost:4317",
		Insecure:    true, // Use insecure connection for local development
		LogLevel:    gintelemetry.LevelInfo,
	})
	if err != nil {
		panic(err)
	}
	defer tel.Shutdown(ctx)

	// Initialize database
	db, err := initDB(ctx, tel)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// List users endpoint
	router.GET("/users", func(c *gin.Context) {
		ctx := c.Request.Context()

		users, err := getUsers(ctx, tel, db)
		if err != nil {
			tel.Log().Error(ctx, "failed to fetch users", "error", err.Error())
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		tel.Log().Info(ctx, "fetched users", "count", len(users))
		tel.Metric().RecordGauge(ctx, "users.count", int64(len(users)))

		c.JSON(200, users)
	})

	// Get user by ID endpoint
	router.GET("/users/:id", func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		user, err := getUserByID(ctx, tel, db, id)
		if err != nil {
			tel.Log().Error(ctx, "failed to fetch user", "user_id", id, "error", err.Error())
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if user == nil {
			c.JSON(404, gin.H{"error": "user not found"})
			return
		}

		tel.Log().Info(ctx, "fetched user", "user_id", id)
		c.JSON(200, user)
	})

	// Create user endpoint
	router.POST("/users", func(c *gin.Context) {
		ctx := c.Request.Context()

		var input struct {
			Name  string `json:"name" binding:"required"`
			Email string `json:"email" binding:"required"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		userID, err := createUser(ctx, tel, db, input.Name, input.Email)
		if err != nil {
			tel.Log().Error(ctx, "failed to create user", "error", err.Error())
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		tel.Log().Info(ctx, "user created", "user_id", userID)
		tel.Metric().IncrementCounter(ctx, "users.created",
			tel.Attr().String("status", "success"),
		)

		c.JSON(201, gin.H{"id": userID})
	})

	tel.Log().Info(ctx, "server starting", "port", 8080)
	router.Run(":8080")
}

func initDB(ctx context.Context, tel *gintelemetry.Telemetry) (*sql.DB, error) {
	return tel.WithSpan(ctx, "db.init", func(ctx context.Context) (*sql.DB, error) {
		tel.Log().Info(ctx, "initializing database")

		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			return nil, err
		}

		// Create schema with timing
		err = tel.MeasureDuration(ctx, "db.schema.create", func() error {
			_, err := db.ExecContext(ctx, `
				CREATE TABLE users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL,
					email TEXT NOT NULL UNIQUE,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)
			`)
			return err
		})
		if err != nil {
			return nil, err
		}

		// Insert sample data
		err = insertSampleData(ctx, tel, db)
		if err != nil {
			return nil, err
		}

		tel.Log().Info(ctx, "database initialized")
		return db, nil
	})
}

func insertSampleData(ctx context.Context, tel *gintelemetry.Telemetry, db *sql.DB) error {
	return tel.WithSpan(ctx, "db.seed", func(ctx context.Context) error {
		users := []struct{ name, email string }{
			{"Alice Smith", "alice@example.com"},
			{"Bob Johnson", "bob@example.com"},
			{"Carol Williams", "carol@example.com"},
		}

		for _, u := range users {
			err := tel.MeasureDuration(ctx, "db.insert.duration", func() error {
				_, err := db.ExecContext(ctx,
					"INSERT INTO users (name, email) VALUES (?, ?)",
					u.name, u.email,
				)
				return err
			})
			if err != nil {
				return err
			}
		}

		tel.Log().Info(ctx, "sample data inserted", "count", len(users))
		return nil
	})
}

func getUsers(ctx context.Context, tel *gintelemetry.Telemetry, db *sql.DB) ([]User, error) {
	ctx, stop := tel.Trace().StartSpanWithAttributes(ctx, "db.query",
		tel.Attr().String("db.operation", "SELECT"),
		tel.Attr().String("db.table", "users"),
	)
	defer stop()

	var users []User

	err := tel.MeasureDuration(ctx, "db.query.duration", func() error {
		rows, err := db.QueryContext(ctx, "SELECT id, name, email, created_at FROM users")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var u User
			if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
				return err
			}
			users = append(users, u)
		}

		return rows.Err()
	})

	if err != nil {
		tel.Trace().RecordError(ctx, err)
		return nil, err
	}

	tel.Trace().SetAttributes(ctx,
		tel.Attr().Int("db.rows_returned", len(users)),
	)

	tel.Metric().IncrementCounter(ctx, "db.queries.total",
		tel.Attr().String("operation", "SELECT"),
		tel.Attr().String("table", "users"),
		tel.Attr().String("status", "success"),
	)

	return users, nil
}

func getUserByID(ctx context.Context, tel *gintelemetry.Telemetry, db *sql.DB, id string) (*User, error) {
	ctx, stop := tel.Trace().StartSpanWithAttributes(ctx, "db.query",
		tel.Attr().String("db.operation", "SELECT"),
		tel.Attr().String("db.table", "users"),
		tel.Attr().String("user.id", id),
	)
	defer stop()

	var user User

	err := tel.MeasureDuration(ctx, "db.query.duration", func() error {
		return db.QueryRowContext(ctx,
			"SELECT id, name, email, created_at FROM users WHERE id = ?",
			id,
		).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	})

	if err == sql.ErrNoRows {
		tel.Log().Warn(ctx, "user not found", "user_id", id)
		return nil, nil
	}

	if err != nil {
		tel.Trace().RecordError(ctx, err)
		return nil, err
	}

	tel.Metric().IncrementCounter(ctx, "db.queries.total",
		tel.Attr().String("operation", "SELECT"),
		tel.Attr().String("table", "users"),
		tel.Attr().String("status", "success"),
	)

	return &user, nil
}

func createUser(ctx context.Context, tel *gintelemetry.Telemetry, db *sql.DB, name, email string) (int64, error) {
	ctx, stop := tel.Trace().StartSpanWithAttributes(ctx, "db.insert",
		tel.Attr().String("db.operation", "INSERT"),
		tel.Attr().String("db.table", "users"),
	)
	defer stop()

	var result sql.Result

	err := tel.MeasureDuration(ctx, "db.insert.duration", func() error {
		var err error
		result, err = db.ExecContext(ctx,
			"INSERT INTO users (name, email) VALUES (?, ?)",
			name, email,
		)
		return err
	})

	if err != nil {
		tel.Trace().RecordError(ctx, err)
		tel.Metric().IncrementCounter(ctx, "db.queries.total",
			tel.Attr().String("operation", "INSERT"),
			tel.Attr().String("table", "users"),
			tel.Attr().String("status", "error"),
		)
		return 0, err
	}

	id, _ := result.LastInsertId()

	tel.Trace().SetAttributes(ctx,
		tel.Attr().Int64("user.id", id),
	)

	tel.Metric().IncrementCounter(ctx, "db.queries.total",
		tel.Attr().String("operation", "INSERT"),
		tel.Attr().String("table", "users"),
		tel.Attr().String("status", "success"),
	)

	return id, nil
}

// WithSpan is a helper that returns a value instead of just error
func (t *gintelemetry.Telemetry) WithSpan[T any](ctx context.Context, spanName string, fn func(context.Context) (T, error)) (T, error) {
	newCtx, stop := t.Trace().StartSpan(ctx, spanName)
	defer stop()

	result, err := fn(newCtx)
	if err != nil {
		t.Trace().RecordError(newCtx, err)
	}

	return result, err
}
