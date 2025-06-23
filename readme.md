# ELORM

The Golang ORM for crafting elegant, database-powered applications.

## Ideas under the hood

Elorm implements set of ideas from my business application engineer experience working:
- Use **global unique human-readable sequental ID** for records/entities. This ID should be globally unique, sequential across all used databases, and should act as a URL for a particular entity. For better handling of database indexes ID should be sortable in order or creation new IDs line autoincrement numbers.
- **Define the set of entities declaratively**, including navigation properties. Its profilable when you context is growing.
- **Lazy load navigation properties**. It retrieves a referenced record on first access to the navigation property, from cache or from the database. I can having a lot of navigation properties without impact of performance. 
- Let the library **handle migrations automatically**. It should be possible to upgrade from any old version to any new one. Developers don't need to register each schema change as a separate migration. And, of course, developers can run their own code as part of this migration.
- Create a common **core entity with basic functionality (loading, saving, caching, etc.)**. Application entities must be based on it.
- Create a **global entity cache** to track all loaded/created entities and reduce redundant queries to database. Of course, we can tune cache size to balance betwee speed and memory for application.
- **Generate strongly-typed structs** with all needed methods based on schema declaration. It should eliminate boilerplate in application code. Entities are ready to use after a couple of lines of code.
- Allow developers to **use the "idiomatic database/sql way"** to work with data. Engineer can use regular SQL select queries.

## Implementation

- Elorm allows you to declare your data schema using JSON schema (with automatic validation). [Elorm-gen](https://github.com/softilium/elorm-gen) compiles it to a strongly-typed set of structs with methods. It also creates a factory struct and methods for operating with data: loading, creating new entities, caching entitiies and handling database structures. 
- Supported databases:
	- PostgreSQL
	- SQLite
	- MySQL
	- MS-SQL

### Factory

- As s start point we need to create a "Entities factory" or just "factory". What factory does:
	- Holds entity types list
	- Handles database migrations automatically. Creates all needed structures inside database and reflects any further entity schema changes.
	- Processes entities: loading, caching and creating new ones.
- Factory has global property named AggressiveReadingCache. By default it is false and factory checks dataversion between cache and database when load cached entity. When you set AggressiveReadingCache to true factory will use data from cache without checking DataVersion before loading from cache. It significally increases performance but you can receive some errors when you try to save outdated (changed by another factories) entities when save.

### Entity type

- Entity type defined object name (singular) and table name (plural). These names used for database structures creation.
- Entity type defines field list. We can define string, int, bool, numeric, and reference (navigation) fields. Each field will be automatically decorated with getters/setters for easy, strongly-typed access from code.
- Entity type defined special Wrap() function. It allows us to wrap generic entity into strongly-typed entity for application purposes, see elorm-gen.

### Entity

- Entities (records) contain some built-in fields such as Ref (Primary key, "ID"), IsDeleted (bool), and DataVersion. Each entity can be loaded, modified, and saved. All entities are cached automatically at the factory level.
- Entity ID. Elorm uses something like special kind of GUIDs. Elorm uses a special format for entity primary key values. Consider it as a "entity URL". Reasons:
	- Having a globally unique and immutable value for each entity from its creation time. Even before saving entity, we can reference it from other entities.
	- Having sequential values (important for primary key indexes performance for most databases)
	- Having a global entity cache (Ref has enough info to load the entity from the database)
	- Ability to use any SQL statements with complex column transformations, table joins, etc. We can determine and load data as entities from result rows based on ref information.
- Entities can be retrieved using standard database/SQL queries. Elorm implements the database/sql Scanner interface.
- Fields store old values alongside the current ones. This is useful when you need to compare values: old and new.
- DataVersion checking prevents changes from concurrent users and can detect and show conflicting changes. Using 
DataVersion is optional at the entity definition level. 

### Elorm-gen

- [Elorm-gen](https://github.com/softilium/elorm-gen) processes JSON with entity schema declarations and generates ready-to-use db-context. It is enriched version of "Factory" with application-scope entity types, getters/setters for fields, etc.
- It dramatically decreases application boilerplate code and makes our code strongly-typed.
- Alongside with created db-context we can define event handlers for it, such as fillNew, beforeSave, afterSave events.
