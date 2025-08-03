# ELORM

The Golang ORM for crafting elegant, database-powered applications.

## Ideas Under the Hood

Elorm implements a set of ideas from my business application engineering experience:

- **Globally unique, human-readable sequential ID** for records/entities. This ID should be globally unique, sequential across all used databases, and should act as a URL for a particular entity. For better handling of database indexes, the ID should be sortable in order of creation, like autoincrement numbers.
- **Define all entities declaratively** using a JSON entity declaration. elorm-gen processes it into strongly-typed structs and methods, ready to use in applications.
- **Shared parts of entities: fragments**. Set of fields, indexes and event handlers. Each entity can include any count of fragments and inherit fields, indexes and event handlers.
- **Handle migrations automatically**. It should be possible to upgrade from any version to any other version. Developers don't need to register each schema change as a separate migration. Of course, developers can run their own code as part of this migration. It works for both table and index definitions.
- **Core entity with basic functionality (loading, saving, caching, etc.)**. Application entities must be based on it.
- **Lazy-load navigation properties**. It retrieves a referenced record on first access to the navigation property, from cache or from the database. You can have many navigation properties without impacting performance.
- **Global entity cache** to track all loaded/created entities and reduce redundant queries to the database. Of course, you can tune cache size to balance between speed and memory for your application.
- **Use the standard database/sql** to work with data. Engineers can use regular SQL select queries as well as specially designed methods.
- **Generate a standard REST API** for each entity type. It should handle CRUD operations as well as grid/table operations (filtering, paging, sorting).

## Get started

### Define entities in JSON

We use standard json schema validation defined here: https://github.com/softilium/elorm-gen/blob/master/elorm.schema.json

Each entity consist of fields and indexes.

Also you can define fragment: set of fields and indexes to reuse between entities. Later you can define event handler routenes for fragment as for entity type.

<details>
<summary>sample.schema.json (fragment)</summary>

```json
{
    "PackageName": "main",
    "Fragments": [
        {
            "FragmentName": "BusinessObjects",
            "Columns": [
                {
                    "Name": "CreatedBy",
                    "ColumnType": "ref.User"
                },
                {
                    "Name": "CreatedAt",
                    "ColumnType": "datetime"
                },
                {
                    "Name": "ModifiedBy",
                    "ColumnType": "ref.User"
                },
                {
                    "Name": "ModifiedAt",
                    "ColumnType": "datetime"
                },
                {
                    "Name": "DeletedBy",
                    "ColumnType": "ref.User"
                },
                {
                    "Name": "DeletedAt",
                    "ColumnType": "datetime"
                }
            ]
        }
    ],
    "Entities": [
        {
            "ObjectName": "User",
            "TableName": "Users",
            "Indexes": [
                {
                    "Columns": [
                        "Username"
                    ],
                    "Unique": true
                },
                {
                    "Columns": [
                        "Email"
                    ],
                    "Unique": true
                }
            ],
            "Columns": [
                {
                    "Name": "Username",
                    "ColumnType": "string",
                    "Len": 100
                },
                {
                    "Name": "Email",
                    "ColumnType": "string",
                    "Len": 100
                },
                {
                    "Name": "PasswordHash",
                    "ColumnType": "string",
                    "Len": 200
                },
                {
                    "Name": "IsActive",
                    "ColumnType": "bool"
                },
                {
                    "Name": "ShopManager",
                    "ColumnType": "bool"
                },
                {
                    "Name": "Admin",
                    "ColumnType": "bool"
                },
                {
                    "Name": "TelegramUsername",
                    "ColumnType": "string",
                    "Len": 100
                },
                {
                    "Name": "TelegramCheckCode",
                    "ColumnType": "string",
                    "Len": 100
                },
                {
                    "Name": "TelegramVerified",
                    "ColumnType": "bool"
                },
                {
                    "Name": "TelegramChatId",
                    "ColumnType": "int"
                },
                {
                    "Name": "Description",
                    "ColumnType": "string",
                    "Len": 300
                }
            ]
        },
        {
            "ObjectName": "Token",
            "TableName": "Tokens",
            "Indexes": [
                {
                    "Columns": [
                        "User"
                    ],
                    "Unique": false
                },
                {
                    "Columns": [
                        "AccessToken",
                        "AccessTokenExpiresAt"
                    ],
                    "Unique": true
                },
                {
                    "Columns": [
                        "RefreshToken",
                        "RefreshTokenExpiresAt"
                    ],
                    "Unique": true
                }
            ],
            "Columns": [
                {
                    "Name": "User",
                    "ColumnType": "ref.User"
                },
                {
                    "Name": "AccessToken",
                    "ColumnType": "string",
                    "Len": 50
                },
                {
                    "Name": "AccessTokenExpiresAt",
                    "ColumnType": "datetime"
                },
                {
                    "Name": "RefreshToken",
                    "ColumnType": "string",
                    "Len": 50
                },
                {
                    "Name": "RefreshTokenExpiresAt",
                    "ColumnType": "datetime"
                }
            ]
        },
        {
            "ObjectName": "Shop",
            "TableName": "Shops",
            "Fragments": [
                "BusinessObjects"
            ],
            "Columns": [
                {
                    "Name": "Caption",
                    "ColumnType": "string",
                    "Len": 100
                },
                {
                    "Name": "Description",
                    "ColumnType": "string",
                    "Len": 300
                },
                {
                    "Name": "DeliveryConditions",
                    "ColumnType": "string",
                    "Len": 300
                }
            ]
        },
        {
            "ObjectName": "Good",
            "TableName": "Goods",
            "Fragments": [
                "BusinessObjects"
            ],
            "Indexes": [
                {
                    "Columns": [
                        "OwnerShop"
                    ],
                    "Unique": false
                }
            ],
            "Columns": [
                {
                    "Name": "OwnerShop",
                    "ColumnType": "ref.Shop"
                },
                {
                    "Name": "Caption",
                    "ColumnType": "string",
                    "Len": 100
                },
                {
                    "Name": "Article",
                    "ColumnType": "string",
                    "Len": 50
                },
                {
                    "Name": "Url",
                    "ColumnType": "string",
                    "Len": 500
                },
                {
                    "Name": "Description",
                    "ColumnType": "string",
                    "Len": 4096
                },
                {
                    "Name": "Price",
                    "ColumnType": "numeric",
                    "Precision": 10,
                    "Scale": 2
                },
                {
                    "Name": "OrderInShop",
                    "ColumnType": "int"
                }
            ]
        }
	}
}

```

</details>


### Run elorm-gen to generate code

Install elorm-gen:

```
go get github.com/softilium/elorm-gen
go install github.com/softilium/elorm-gen
```

run elorm-gen for your json on your project folder:

```
elorm-gen sample.schema.json dbcontext-sample.go
```

dbcontext-sample.go should contain generated structs for your entities from sample.schema.json. It also contains methods for operating with data: loading, creating new entities, caching entities, and handling database structures.

By default generated file uses package name "main", but you can redefine in on JSON or on command line:

```
elorm-gen sample.schema.json dbcontext-sample.go --package m2
```

### Initialize elorm on your app


Add "elorm" library to your project using:

```
go get github.com/softilium/elorm
```

Initialize db-context with your entities:

```go

	dialect := "sqlite" // postgres, mssql, mysql also supported

	// syntax of connection string are defined by corresponding database driver
	connectionString := "file:todo.db?cache=shared"

	DB, err = CreateDbContext(dialect, connectionString)
	logError(err)

	// in case we have only one DB instance for database we can use aggressive caching strategy
	DB.AggressiveReadingCache = true

```

After initialized db-context can be enriched using additional event handlers. For example:

```go
	err = DB.AddFillNewHandler(dbc.UserDef.EntityDef, func(entity any) error {
		user := entity.(*User)
		user.SetIsActive(true)
		return nil
	})
	if err != nil {
		return err
	}
```

Before first usage we need to ensure database has all tables and indexes ready to store data:

```go
	err = DB.EnsureDBStructure()
	logError(err)
	fmt.Println("Database structure ensured successfully.")
```

After that your DB is ready to work with entitites.

### Work with entities

Right after initialization DB could be used to process entities. Lets seed users table on first start and remove expired tokens:

```go


	// create default admin on empty users table

	users, _, err := DB.UserDef.SelectEntities(nil, nil, 0, 0)
	if err != nil {
		logError(err)
	}
	if len(users) == 0 {

		admPassword := "<sample-password>"
		admUserName := "sample@gmail.com"

		fmt.Println("No users found, creating default admin user...")
		adminUser, err := DB.CreateUser()
		if err != nil {
			logError(err)
		}
		admPwdHash, _ := HashPassword(admPassword)
		adminUser.SetEmail(admUserName)
		adminUser.SetUsername(admUserName)
		adminUser.SetPasswordHash(admPwdHash)
		adminUser.SetAdmin(true)
		err = adminUser.Save(context.Background())
		if err != nil {
			logError(err)
		}
	}

	// remove expired tokens on start
	tokensToDelete, _, err := DB.TokenDef.SelectEntities([]*elorm.Filter{
		elorm.AddFilterLT(DB.TokenDef.RefreshTokenExpiresAt, time.Now()),
	}, nil, 0, 0)
	if err != nil {
		logError(err)
	} else {
		for _, token := range tokensToDelete {
			err = DB.DeleteEntity(context.Background(), token.RefString())
			if err != nil {
				logError(err)
			}
		}
		fmt.Printf("Deleted %d expired tokens\n", len(tokensToDelete))
	}

```

Pay an attention all methods such as Save(), LoadEntity(), DeleteEntity() take ctx parameter to pass value to event handlers.

### Create REST API from entities

(to be done)

### JSON marshaling

(to be done)

### Using standard Go idiomatic approaches

database/sql and contexts

(to be done)

### Define and redefine entities on the fly

(to be done)

### DataVersion checking and AggressiveCaching

(to be done)

### old field values, easy work with changes

(to be done)

### Caching entities, lazy-loading reference (navigation) properties

(to be done)
