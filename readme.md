# ELORM

The Golang ORM for crafting elegant, database-powered applications.

## Ideas Under the Hood

Elorm implements a set of ideas from my business application engineering experience:

- **Globally unique, human-readable sequential ID** for records/entities. This ID should be globally unique, sequential across all used databases, and should act as a URL for a particular entity. For better handling of database indexes, the ID should be sortable in order of creation, like autoincrement numbers.
- **Define all entities declaratively** using a JSON entity declaration. elorm-gen processes it into strongly-typed structs and methods, ready to use in applications.
- **Shared parts of entities - fragments**. Set of fields, indexes and event handlers. Each entity can include any cound of fragments and inherid fields, indexes and event handlers.
- **Handle migrations automatically**. It should be possible to upgrade from any version to any. Developers don't need to register each schema change as a separate migration. Of course, developers can run their own code as part of this migration. It works for both tables and indexes definitions.
- **Core entity with basic functionality (loading, saving, caching, etc.)**. Application entities must be based on it.
- **Lazy load navigation properties**. It retrieves a referenced record on first access to the navigation property, from cache or from the database. You can have many navigation properties without impacting performance.
- **Global entity cache** to track all loaded/created entities and reduce redundant queries to the database. Of course, you can tune cache size to balance between speed and memory for your application.
- **Use the standard database/sql** to work with data. Engineers can use regular SQL select queries as well as specially designed methods.
- **Generate a standard REST API** for each entity type. It should handle CRUD operations as well as grid/table operations (filtering, paging, sorting).

## Implementation

- Elorm allows you to declare your data schema using JSON schema (with automatic validation). [Elorm-gen](https://github.com/softilium/elorm-gen) compiles it to a strongly-typed set of structs with methods. It also creates a factory struct and methods for operating with data: loading, creating new entities, caching entities, and handling database structures.
- Supported databases:
	- PostgreSQL
	- SQLite
	- MySQL
	- MS-SQL

### Factory

- As a starting point, you need to create an "Entities factory" or just "factory". The factory:
	- Holds the list of entity types
	- Handles database migrations automatically. It creates all needed structures inside the database and reflects any further entity schema changes.
	- Processes entities: loading, caching, and creating new ones.
- The factory has a global property named AggressiveReadingCache. By default, it is false and the factory checks the DataVersion between the cache and database when loading a cached entity. When you set AggressiveReadingCache to true, the factory will use data from the cache without checking DataVersion before loading from cache. This significantly increases performance, but you may encounter errors if you try to save outdated (changed by another factory) entities.

### Entity Type

- An entity type defines the object name (singular) and table name (plural). These names are used for database structure creation.
- An entity type defines a field list. You can define string, int, bool, numeric, and reference (navigation) fields. Each field is automatically decorated with getters/setters for easy, strongly-typed access from code.
- An entity type defines a special Wrap() function. It allows you to wrap a generic entity into a strongly-typed entity for application purposes (see elorm-gen).

### Entity

- Entities (records) contain some built-in fields such as Ref (Primary key, "ID"), IsDeleted (bool), and DataVersion. Each entity can be loaded, modified, and saved. All entities are cached automatically at the factory level.
- Entity ID: Elorm uses a special kind of GUID for entity primary key values. Consider it as an "entity URL". Reasons:
	- Having a globally unique and immutable value for each entity from its creation time. Even before saving the entity, we can reference it from other entities.
	- Having sequential values (important for primary key index performance for most databases)
	- Having a global entity cache (Ref has enough info to load the entity from the database)
	- Ability to use any SQL statements with complex column transformations, table joins, etc. We can determine and load data as entities from result rows based on ref information.
- Entities support JSON marshalling and unmarshalling.
- Entities can be retrieved using standard database/SQL queries. Elorm implements the database/sql Scanner interface.
- Fields store old values alongside the current ones. This is useful when you need to compare old and new values.
- DataVersion checking prevents changes from concurrent users and can detect and show conflicting changes. Using DataVersion is optional at the entity definition level.

### Elorm-gen

- [Elorm-gen](https://github.com/softilium/elorm-gen) processes JSON with entity schema declarations and generates a ready-to-use db-context. It is an enriched version of "Factory" with application-scope entity types, getters/setters for fields, etc.
- It dramatically decreases application boilerplate code and makes your code strongly-typed.
- Alongside the created db-context, you can define event handlers for it, such as fillNew, beforeSave, and afterSave events.
