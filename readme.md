# ELORM

The Golang ORM for crafting elegant, database-powered applications.

[![Go Reference](https://pkg.go.dev/badge/github.com/softilium/elorm.svg)](https://pkg.go.dev/github.com/softilium/elorm)

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
- **Generate a standard REST API** for each entity type. It should handle CRUD operations as well as grid/table operations (filtering, paging, sorting). Also, entities support JSON serialization out of the box.
- **Soft delete mode for entities** allows us to transparently mark entities as deleted without deleting them from the database with minimum effort.
- **Optimistic locks** allow a safe way for multi-user working with databases.

## Quick start with ELORM

### Define entities in JSON

We use standard json schema validation defined here: https://github.com/softilium/elorm-gen/blob/master/elorm.schema.json

Each entity consists of fields and indexes.

Also, you can define fragments: sets of fields and indexes to reuse between entities. Later you can define event handler routines for fragments as for entity types.

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

By default generated file uses package name "main", but you can redefine it in JSON or on command line:

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

Right after initialization DB could be used to process entities. Let's seed users table on first start and remove expired tokens:

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

Pay attention that all methods such as Save(), LoadEntity(), DeleteEntity() take ctx parameter to pass value to event handlers.

### Create REST API from entities

ELORM allows you to create standard HTTP REST APIs for entities. Filtering, sorting and paging are supported out of the box. Let's look at an example:

```go
	// first define config of Api element
	goodsRestApiConfig := elorm.CreateStdRestApiConfig(
		*DB.GoodDef.EntityDef,
		DB.LoadGood,
		DB.GoodDef.SelectEntities,
		DB.CreateGood)

	// define additional filter. This filter should always be merged with user-defined filters
	goodsRestApiConfig.AdditionalFilter = func(r *http.Request) ([]*elorm.Filter, error) {
		res := []*elorm.Filter{}
		res = append(res, elorm.AddFilterGT(DB.GoodDef.Price 10))
		shopref := r.URL.Query().Get("shopref")
		if shopref != "" {
			res = append(res, elorm.AddFilterEQ(DB.GoodDef.OwnerShop, shopref))
		}
		return res, nil
	}
	// define default sort order. User can define it's own using query parameter
	goodsRestApiConfig.DefaultSorts = func(r *http.Request) ([]*elorm.SortItem, error) {
		return []*elorm.SortItem{{Field: DB.GoodDef.OrderInShop, Asc: true}}, nil
	}
	// Context func creates all needed values to pass to event handlers
	goodsRestApiConfig.Context = LoadUserFromHttpToContext

	//here we connect handler to end-point on standard http router 
	router.HandleFunc("/api/goods", elorm.HandleRestApi(goodsRestApiConfig))
```

See more about RestApiConfig: 
https://pkg.go.dev/github.com/softilium/elorm#RestApiConfig 

### Soft delete for entities

Entity definition supports UseSoftDelete mode. By default, UseSoftDelete is false.

In this mode ELORM adds filter "IsDeleted=true" to SelectEntities() filter unless developer adds his own filter on "IsDeleted".
Also REST API handles DELETE requests as "let's mark entity as IsDelete=true" instead of deleting it.

In default mode (UseSoftDelete=false) Save() method returns an error is we set IsDeleted to true.

Note. Each entity has IsDeleted field, UseSoftDelete=false doesn't remove the field.

### Use standard Go idiomatic approaches

ELORM stay on top on best Go idiomatic code approaches when it is possible and as many as it possible.

#### database/sql

Developers can use standard "database/sql" approach to retrieve data and ELORM can enrich Scanner interface:

```go
	rows, err := DB.Query("select ref from goods order by ref")
	checkErr(err)
	defer rows.Close()
	idx := 0
	for rows.Next() {

        var ref string
        rows.Scan(&ref)
		checkErr(err)

		// access to row column as typed Entity
		loadedGood, err := DB.LoadGood(ref)
		checkErr(err)

		// access to lazy-loading property CreatedBy() (User) and its property Username
		fmt.Println(loadedGood.CreatedBy().Username())
	}
```
#### context

Standard Go context is used to pass any additional parameters to event handlers:

```go
	// let's define before save handler for any entity with reference to BusinessObject fragment (Shop, Good)
	err := DB.AddBeforeSaveHandler(BusinessObjectsFragment, func(ctx context.Context, entity any) error {
		user, ok := ctx.Value(userContextKey).(*User)
		if !ok {
			user = nil
		}
		et := entity.(BusinessObjectsFragmentMethods)
		ent := entity.(elorm.IEntity)
		if ent.IsNew() {
			et.SetCreatedAt(time.Now())
			if user != nil {
				et.SetCreatedBy(user)
			}
			if et.CreatedBy() == nil {
				return fmt.Errorf("createdBy is required for new entity %s", ent.Def().ObjectName)
			}
		} else {
			et.SetModifiedAt(time.Now())
			if user != nil {
				et.SetModifiedBy(user)
			}
		}
	})

	AddUserContext := func (ctx context.Context, user *User) context.Context {
		if user != nil {
			ctx = context.WithValue(ctx, userContextKey, user)
		}
		return ctx
	}

	// get first user as Current from database
	CurrentUser := DB.UserDef.SelectEntities(nil, nil, 1, 1)
	saveCtx := AddUserContext(context.Background())
	NewGood, err := DB.CreateGood()
	if err != nil {
		logError(err)
	}

	//save new Good, NewGood.CreatedBy should be initialized by CurrentUser via event handler
	err = NewGood.Save(saveCtx)
	if err != nil {
		logError(err)
	}

```

#### JSON marshaling/unmarchaling

By default, any entity can be serialized to JSON and deserialized from JSON. Standard Marshaler/Unmarshaler interfaces are implemented. 

Occasionally, we need to "expand" some references (navigation properties). For example, it is a common approach to serialize user not just as user ID but as object with base properties: refID, userName, email, etc.

Elorm supports automatic references expanding for JSON when you need it. Use AutoExpandFieldsForJSON property for Entity Definition:

```go
	dbc.UserDef.AutoExpandFieldsForJSON = map[*elorm.FieldDef]bool{
		dbc.UserDef.Ref:      true,
		dbc.UserDef.Username: true,
	}
```

After that all fields that reference User should be expanded to Ref, Username when you serialize entity to JSON.

### DataVersion checking and AggressiveCaching

We use typical optimistic locks implementation in ELORM. DataVersion field is assigned to new value before each save. And if it is defined on factory and entity def levels, we check that actual database version contains old value of DataVersion field. If it contains a different value, it means another client has updated row in database after we read it last time. In that case Save() returns an error.

### Work with old field values

When we load entity and change values for some fields then old values are accessible via Old() methods before we saved entity to database. It is useful to analyze changes in BeforeSave handlers.

### Cache entities, lazy-loading reference (navigation) properties

All loaded entities are cached at the factory level using LRU cache. Next LoadEntity() will load entity from cache instead of querying the database. Our internal tests show that it increases speed of loading entities by about 100 times. Developers don't need to worry about cache or do anything to maintain it. 

### Handling Transactions

Factory struct created and holds database connector (*sql.DB) based on dialect and connection string parameters. 

ELORM handles transactions on all standard operations such as save or delete. BeforeSave event handler works within main transaction and when handler returns an error, save transactions will be rolled back.

AfterSave event handler works after main transaction is committed.

Developers don't need to start a transaction before saving or deleting entities. But when you need a transaction to wrap some actions into it, the recommended approach is:

```go
		tx, err := DB.BeginTran()
		if err != nil {
			HandleErr(err)
			return
		}
		defer func() { _ = DB.RollbackTran(tx) }()

		old, _, err := DB.GoodTagDef.SelectEntities(
			[]*elorm.Filter{elorm.AddFilterEQ(DB.GoodTagDef.Good, good)}, nil, 0, 0)
		if err != nil {
			HandleErr(err)
			return
		}
		for _, ot := range old {
			err = DB.DeleteEntity(r.Context(), ot.RefString())
			if err != nil {
				HandleErr(err)
				return
			}
		}
		for _, line := range result {
			if line.Tagged {
				gt, err := DB.CreateGoodTag()
				gt.SetGood(good)
				gt.SetTag(tg)
				err = gt.Save(r.Context())
				if err != nil {
					HandleErr(err))
					return
				}
			}
		}
		err = DB.CommitTran(tx)
		if err != nil {
			HandleErr(err))
			return
		}
```

### SQLite and multithreading

Let's use simple example to explain.

So, we have two entity types: Order and OrderLine. Order owns some Order lines and order lines should be deleted before order is deleted. Typically, we implement BeforeDelete handler for Order to delete OrderLines. Each DeleteEntity has its own transaction and we want to rollback all transactions when any is rolled back. For most databases it is implemented by different *sql.Tx transactions. Except SQLite.

SQLite doesn't support more than one writing transaction at the same time. Because of this we use different approach for "nested" transactions on SQLite. We start first transaction as usual. But second transaction doesn't really start on SQLite. We just increase transaction level counter. When we commit transaction we decrease that counter. If counter goes to zero, we issue "Commit" to SQLite. It allows us to emulate "nested" transactions behavior for SQLite. 

Dark side of this solution is that we need to stay away from goroutines when we use SQLite. One goroutine can break "transaction counter" for another goroutine.

We fully support goroutines and multithreading in ELORM for MySQL, Postgres and Microsoft SQL databases.

### Fragments

Fragments are just sets of columns, indexes and event handlers. Fragment is defined in JSON using similar way as entity. Later, entities can reference a fragment and all columns from fragment should be copied to entity. We can describe some set of "functionality" and then reuse it many times in many entities. This makes our code compact and easy to understand.

Note, we still use strongly-typed entities and when we remove field from fragment definition, compiler will show all removed field usage lines.

### Wrapping entities into strongly-typed structs with methods

Each entity type definition can assign special function that converts "universal" entity into strongly typed one. Strongly typed entity can define methods to access fields, additional service methods, etc. elorm-gen uses this functionality to automatically create strongly typed entities for you based on JSON entity definitions.
