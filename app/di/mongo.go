package di

import (
	"context"
	"fmt"

	"github.com/open-source-cloud/fuse/app/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// provideMongoClient provides a MongoDB client if the driver is MongoDB, otherwise returns nil
func provideMongoClient(cfg *config.Config) *mongo.Client {
	if cfg.Database.Driver != mongodbDriver && cfg.Database.Driver != mongoDriver {
		return nil
	}

	// Build MongoDB connection string
	connectionString := fmt.Sprintf("mongodb://%s:%s/%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)

	if cfg.Database.User != "" && cfg.Database.Pass != "" {
		connectionString = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s",
			cfg.Database.User,
			cfg.Database.Pass,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.Name,
		)
	}

	clientOptions := options.Client().ApplyURI(connectionString)

	// Handle TLS configuration
	if cfg.Database.TLS {
		clientOptions = clientOptions.SetTLSConfig(nil) // Use default TLS config
	}

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MongoDB: %v", err))
	}

	// Test the connection
	ctx := context.Background()
	if err := client.Ping(ctx, nil); err != nil {
		panic(fmt.Sprintf("Failed to ping MongoDB: %v", err))
	}

	return client
}
