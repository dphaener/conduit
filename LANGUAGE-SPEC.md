# Conduit Language Specification

**Version:** 2.0
**Status:** Ready for Implementation
**Updated:** 2025-10-13

This document is the canonical reference for the Conduit programming language. It defines the complete syntax, semantics, type system, standard library, and expression language.

---

## Table of Contents

1. [Overview & Philosophy](#overview--philosophy)
2. [Type System](#type-system)
3. [Resource Syntax](#resource-syntax)
4. [Expression Language](#expression-language)
5. [Standard Library](#standard-library)
6. [Lifecycle Hooks](#lifecycle-hooks)
7. [Validation & Constraints](#validation--constraints)
8. [Error Handling](#error-handling)
9. [Query Language](#query-language)
10. [Complete Examples](#complete-examples)

---

## Overview & Philosophy

### Design Principles

**Explicitness over Brevity**
- LLMs don't experience tedium, they experience ambiguity
- Verbose in service of clarity is GOOD
- Ceremonial verbosity is BAD

**Progressive Disclosure**
- Simple things stay simple (3 lines can be a complete resource)
- Complexity is available when needed
- No forced sophistication

**Correctness over Cleverness**
- Structure carries meaning
- Type safety prevents errors
- Explicit error handling

**Zero Ambiguity**
- Every type must specify nullability (`!` vs `?`)
- All built-in functions are namespaced (`String.slugify()`)
- Transaction boundaries are explicit (`@transaction`, `@async`)
- Custom functions must be defined with `@function`

### File Extension

Conduit files use the `.cdt` extension:
```
user.cdt
post.cdt
product.cdt
```

---

## Type System

### Explicit Nullability

**Every type must specify nullability:**

```
type!     // Required, never null
type?     // Optional, can be null
```

**Examples:**
```
// Strings
username: string!         // Required
bio: text?               // Optional

// Numbers
price: float!            // Required
sale_price: float?       // Optional

// Relationships
author: User!            // Required relationship
category: Category?      // Optional relationship

// Arrays
tags: array<uuid>!       // Required array (can be empty)
images: array<string>?   // Optional array (can be null)
```

**File-level enforcement:**
```
@strict nullability  // All types must use ! or ?
```

### Primitive Types

```
// Text
string!               // Variable-length text
string(50)!          // Max length 50
text!                // Long-form text
markdown!            // Markdown content

// Numbers
int!                 // Integer
float!               // Floating point
decimal(10,2)!       // Decimal with precision

// Boolean
bool!                // true/false

// Time
timestamp!           // Date and time
date!                // Date only
time!                // Time only

// Unique
uuid!                // UUID v4
ulid!                // ULID

// Special
email!               // Email address (validated)
url!                 // URL (validated)
phone!               // Phone number (validated)
json!                // JSON data (escape hatch)
```

### Structural Types

**Arrays:**
```
array<T>!            // Typed array
array<uuid>!         // Array of UUIDs
array<string>!       // Array of strings
array<int>!          // Array of integers
```

**Hashes/Maps:**
```
hash<K, V>!          // Typed hash/map
hash<string, int>!   // String keys, int values
hash<uuid, float>!   // UUID keys, float values
```

**Inline Structs:**
```
// Inline object type
field_name: {
  sub_field: type!
  optional_field: type?
  nested: {
    deep_field: type!
  }
}!

// Example
seo: {
  title: string? @max(60)
  description: string? @max(160)
  image: url?
}?
```

**Nested Arrays:**
```
// Array of structs
images: array<{
  url: string!
  alt: string?
  order: int! @default(0)
}>!

// Array of arrays
matrix: array<array<int>>!
```

### Enum Types

```
status: enum ["draft", "published", "archived"]! @default("draft")
role: enum ["user", "admin", "moderator"]! @default("user")
```

### Default Values

```
// Primitive defaults
count: int! @default(0)
active: bool! @default(true)
status: enum ["active", "inactive"]! @default("active")

// Structural defaults
tags: array<uuid>! @default([])
metadata: hash<string, string>! @default({})
settings: {
  theme: string! @default("light")
  notifications: bool! @default(true)
}! @default({})
```

---

## Resource Syntax

### Basic Structure

```
/// Documentation comment (required)
/// Explains what this resource represents
resource ResourceName {
  // Fields
  field_name: type! @constraint

  // Relationships
  related: OtherResource!

  // Annotations
  @has_many Related as "collection"
  @nested under Parent
  @middleware [list]

  // Custom functions
  @function name(param: type) -> return_type { }

  // Lifecycle hooks
  @before operation { }
  @after operation { }

  // Validation
  @validate { }
  @constraint name { }
  @invariant name { }

  // Computed fields
  @computed name: type { }

  // Middleware
  @on operation: [middleware]

  // Query scopes
  @scope name { }
}
```

### Field Definitions

```
resource User {
  // Primary key
  id: uuid! @primary @auto

  // Simple fields
  username: string! @unique @min(3) @max(50)
  email: email! @unique
  bio: text?

  // Grouped fields (nested objects)
  profile: {
    full_name: string!
    avatar_url: url?
    location: string?
    website: url?
  }!

  // Social links
  social: {
    twitter: string?
    github: string?
    linkedin: string?
  }? @default({})

  // Settings
  preferences: {
    theme: enum ["light", "dark"]! @default("light")
    language: string! @default("en")
    notifications: bool! @default(true)
  }! @default({})

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
  last_login_at: timestamp?
}
```

### Constraints

```
// Field-level
username: string! @unique @min(3) @max(50)
email: email! @unique @required
age: int! @min(18) @max(120)
price: float! @min(0.0)
rating: float! @min(0.0) @max(5.0)
slug: string! @unique @pattern(/^[a-z0-9-]+$/)

// Compound constraints
coordinates: {
  lat: float! @min(-90.0) @max(90.0)
  lng: float! @min(-180.0) @max(180.0)
}!
```

### Relationships

**Belongs To (Foreign Key):**
```
// Simple
author: User!

// With metadata
author: User! {
  foreign_key: "author_id"
  on_delete: restrict          // restrict | cascade | set_null | no_action
  on_update: cascade
}

// Optional
category: Category? {
  foreign_key: "category_id"
  on_delete: set_null
}
```

**Has Many:**
```
@has_many Post as "posts"

// With metadata
@has_many Comment as "comments" {
  foreign_key: "post_id"
  on_delete: cascade
  order_by: "created_at DESC"
}
```

**Has Many Through:**
```
@has_many Tag through PostTag as "tags" {
  join_table: "post_tags"
  foreign_key: "post_id"
  association_foreign_key: "tag_id"
}
```

**Self-Referential:**
```
parent: Category? {
  foreign_key: "parent_id"
  on_delete: set_null
}

@has_many Category as "children" {
  foreign_key: "parent_id"
  on_delete: cascade
}
```

### Nested Resources

```
@nested under Post as "comments"
@operations [list, get, create]  // Limit allowed operations
```

---

## Expression Language

This section defines the expression language used within hooks, validations, computed fields, and custom functions.

### Literals

**String Literals:**
```
"double quoted string"
"with escape sequences: \n \t \\ \""
"multiline strings
can span multiple lines"

// String interpolation
"Hello, #{user.name}!"
"Total: $#{order.total}"
```

**Numeric Literals:**
```
// Integers
42
-17
0
1_000_000    // Underscores for readability

// Floats
3.14
-0.5
2.5e10       // Scientific notation
```

**Boolean and Nil:**
```
true
false
nil          // Represents absence of value
```

**Array Literals:**
```
[]           // Empty array
[1, 2, 3]
["a", "b", "c"]
```

**Hash Literals:**
```
{}           // Empty hash
{name: "Alice", age: 30}
{
  title: "Blog Post",
  author_id: user.id,
  published: true
}
```

### Operators

**Arithmetic:**
```
+    Addition
-    Subtraction
*    Multiplication
/    Division
%    Modulo
**   Exponentiation

// Examples
self.total + self.tax
self.price * self.quantity
2 ** 8  // 256
```

**Comparison:**
```
==   Equal
!=   Not equal
<    Less than
>    Greater than
<=   Less than or equal
>=   Greater than or equal

// Examples
self.age >= 18
self.status == "published"
```

**Logical:**
```
&&   Logical AND (short-circuit)
||   Logical OR (short-circuit)
!    Logical NOT

// Examples
self.age >= 18 && self.email_verified
self.role == "admin" || self.role == "moderator"
!self.deleted_at
```

**String:**
```
+    Concatenation
*    Repetition

// Examples
self.first_name + " " + self.last_name
"-" * 10  // "----------"
```

**Null Coalescing:**
```
??   Return right side if left is nil

// Examples
self.excerpt ?? generate_excerpt(self.content, 200)
self.name ?? "Anonymous"
```

**Safe Navigation:**
```
?.   Only access if not nil

// Examples
self.parent?.name
self.author?.display_name ?? "Unknown"
```

**Membership:**
```
in   Check if value in array or key in hash

// Examples
self.status in ["published", "scheduled"]
"admin" in self.roles
```

### Variables and Field Access

**Self Reference:**
```
self             // Current resource instance
self.field_name  // Access field value
self.author      // Access related resource
self.author.name // Deep property access
```

**Local Variables:**
```
let slug = String.slugify(self.title)
let excerpt = generate_excerpt(self.content, 200)
```

**Field Assignment:**
```
self.field = value
self.published_at = Time.now()
self.view_count = self.view_count + 1
```

### Truthiness

**Falsy Values:**
- `false`
- `nil`
- Empty string `""`
- Zero `0` and `0.0`
- Empty array `[]`
- Empty hash `{}`

**Truthy:** Everything else

### Control Flow

**If Statements:**
```
if condition {
  // body
}

if condition {
  // true branch
} else {
  // false branch
}

if condition1 {
  // branch 1
} elsif condition2 {
  // branch 2
} else {
  // default
}
```

**Unless Statements:**
```
unless condition {
  // executes if condition is FALSE
}
```

**Ternary Operator:**
```
condition ? true_value : false_value

// Example
self.display_name = self.name ? self.name : "Anonymous"
```

**Early Return:**
```
return value    // Exit early from computed field or hook
```

**Match Expression:**
```
let symbol = match self.pricing.currency {
  "USD" => "$",
  "EUR" => "€",
  "GBP" => "£"
}
```

### Function Calls

**Namespaced (Built-in):**
```
String.slugify(self.title)
Text.calculate_reading_time(self.content, words_per_minute: 200)
Time.now()
Array.first(self.items)
```

**Custom Functions:**
```
// Must be defined with @function
generate_slug(self.title)
calculate_discount(self.price, 10)
```

**Method Calls:**
```
self.email.downcase()
self.title.truncate(50)
```

---

## Standard Library

All built-in functions are namespaced to prevent ambiguity.

### String Namespace

```
String.slugify(text: string) -> string
String.capitalize(text: string) -> string
String.upcase(text: string) -> string
String.downcase(text: string) -> string
String.trim(text: string) -> string
String.truncate(text: string, length: int) -> string
String.split(text: string, delimiter: string) -> array<string>
String.join(parts: array<string>, delimiter: string) -> string
String.replace(text: string, pattern: string, replacement: string) -> string
String.starts_with?(text: string, prefix: string) -> bool
String.ends_with?(text: string, suffix: string) -> bool
String.includes?(text: string, substring: string) -> bool
String.length(text: string) -> int
```

### Text Namespace

```
Text.calculate_reading_time(content: text, words_per_minute: int) -> int
Text.word_count(content: text) -> int
Text.character_count(content: text) -> int
Text.excerpt(content: text, length: int) -> string
```

### Number Namespace

```
Number.format(num: float, decimals: int) -> string
Number.round(num: float, precision: int) -> float
Number.abs(num: float) -> float
Number.ceil(num: float) -> int
Number.floor(num: float) -> int
Number.min(a: float, b: float) -> float
Number.max(a: float, b: float) -> float
```

### Array Namespace

```
Array.first<T>(arr: array<T>) -> T?
Array.last<T>(arr: array<T>) -> T?
Array.length<T>(arr: array<T>) -> int
Array.empty?<T>(arr: array<T>) -> bool
Array.includes?<T>(arr: array<T>, item: T) -> bool
Array.unique<T>(arr: array<T>) -> array<T>
Array.sort<T>(arr: array<T>) -> array<T>
Array.reverse<T>(arr: array<T>) -> array<T>
Array.push<T>(arr: array<T>, item: T) -> array<T>
Array.concat<T>(arr1: array<T>, arr2: array<T>) -> array<T>
```

### Hash Namespace

```
Hash.keys<K,V>(hash: hash<K,V>) -> array<K>
Hash.values<K,V>(hash: hash<K,V>) -> array<V>
Hash.merge<K,V>(hash1: hash<K,V>, hash2: hash<K,V>) -> hash<K,V>
Hash.has_key?<K,V>(hash: hash<K,V>, key: K) -> bool
Hash.get<K,V>(hash: hash<K,V>, key: K, default: V?) -> V?
```

### Time Namespace

```
Time.now() -> timestamp
Time.today() -> date
Time.parse(str: string, format: string?) -> timestamp
Time.format(time: timestamp, format: string) -> string
Time.add(time: timestamp, duration: duration) -> timestamp
Time.subtract(time: timestamp, duration: duration) -> timestamp
Time.diff(t1: timestamp, t2: timestamp) -> duration
Time.year(time: timestamp) -> int
Time.month(time: timestamp) -> int
Time.day(time: timestamp) -> int
Time.hour(time: timestamp) -> int
Time.minute(time: timestamp) -> int
Time.second(time: timestamp) -> int
```

### UUID Namespace

```
UUID.generate() -> uuid
UUID.validate(str: string) -> bool
UUID.parse(str: string) -> uuid?
```

### Random Namespace

```
Random.int(min: int, max: int) -> int
Random.float(min: float, max: float) -> float
Random.uuid() -> uuid
Random.hex(length: int) -> string
Random.alphanumeric(length: int) -> string
```

### Crypto Namespace

```
Crypto.hash(data: string, algorithm: string) -> string
Crypto.compare(hash: string, data: string) -> bool
Crypto.encrypt(data: string, key: string) -> string
Crypto.decrypt(data: string, key: string) -> string
```

### HTML Namespace

```
HTML.strip_tags(html: string) -> string
HTML.escape(str: string) -> string
HTML.unescape(str: string) -> string
```

### JSON Namespace

```
JSON.parse(str: string) -> json
JSON.stringify(data: json, pretty: bool?) -> string
JSON.validate(str: string) -> bool
```

### Regex Namespace

```
Regex.match(text: string, pattern: string) -> array<string>?
Regex.replace(text: string, pattern: string, replacement: string) -> string
Regex.test(text: string, pattern: string) -> bool
Regex.split(text: string, pattern: string) -> array<string>
```

### Logger Namespace

```
Logger.debug(message: string, context: hash<string, any>?) -> void
Logger.info(message: string, context: hash<string, any>?) -> void
Logger.warn(message: string, context: hash<string, any>?) -> void
Logger.error(message: string, error: error?, context: hash<string, any>?) -> void
```

### Cache Namespace

```
Cache.get(key: string) -> any?
Cache.set(key: string, value: any, ttl: int?) -> void
Cache.invalidate(key: string) -> void
Cache.clear() -> void
```

### Context Namespace

```
Context.current_user() -> User?
Context.current_user!() -> User  // Unwrap or panic
Context.authenticated?() -> bool
Context.current_request() -> Request
```

### Env Namespace

```
Env.get(key: string, default: string?) -> string?
Env.set(key: string, value: string) -> void
Env.has?(key: string) -> bool
```

### Custom Functions

```
// Define custom functions co-located with resource
@function generate_slug(title: string) -> string {
  let cleaned = String.downcase(title)
  let replaced = Regex.replace(cleaned, /[^a-z0-9]+/, "-")
  return String.trim(replaced, "-")
}

@function calculate_discount(price: float, percent: float) -> float {
  return price * (1.0 - percent / 100.0)
}

// Use in hooks
@before create {
  self.slug = generate_slug(self.title)
}
```

---

## Lifecycle Hooks

### Hook Types

```
@before create { }     // Before insert
@before update { }     // Before update
@before delete { }     // Before delete
@before save { }       // Before create OR update

@after create { }      // After insert
@after update { }      // After update
@after delete { }      // After delete
@after save { }        // After create OR update
```

### Transaction Boundaries

```
// Run in database transaction (failures roll back)
@after create @transaction {
  self.order_number = generate_order_number()!
  OrderItem.create_from_cart(self.cart_items)!
}

// Async operations (non-blocking, failures logged)
@after create @transaction {
  // Critical operations
  self.published_at = Time.now()

  // Non-critical async operations
  @async {
    Notify.send(subscribers, template: "new_post") rescue |err| {
      Logger.error("Failed to notify", error: err)
    }

    SearchIndex.update(self) rescue |err| {
      Logger.warn("Search index failed", error: err)
    }
  }
}
```

### Change Tracking

Available in `@before update` and `@after update` hooks:

```
@after update {
  // Check if field changed
  if self.content_changed? {
    Revision.create!({
      post_id: self.id,
      content: self.previous_value(:content)
    })
  }

  // Check specific value
  if self.status_changed_to?("published") {
    self.published_at = Time.now()
  }

  if self.status_changed_from?("draft") {
    Logger.info("Post #{self.id} moved from draft")
  }
}
```

### Resource Operations

```
// Create
Post.create!({ title: "...", content: "..." })

// Update
Post.update!(id, { title: "..." })

// Delete
Post.delete!(id)

// Increment/Decrement
Post.increment!(id, :view_count)
Post.decrement!(id, :stock_quantity, by: 5)

// Bulk operations
Post.bulk_create!(items)
Post.bulk_update!(updates)
```

---

## Validation & Constraints

### Procedural Validation

```
@validate {
  // Complex logic requiring procedural code
  if self.discount_code {
    let discount = Discount.find_by(code: self.discount_code)
    if !discount || discount.expired? {
      error("Invalid or expired discount code")
    }
  }

  // Multi-field validation
  if self.start_date && self.end_date {
    if self.start_date! >= self.end_date! {
      error("Start date must be before end date")
    }
  }
}
```

### Declarative Constraints

```
@constraint sale_price_valid {
  on: [create, update]
  when: self.pricing.sale != nil
  condition: self.pricing.sale! < self.pricing.regular
  error: "Sale price must be less than regular price"
}

@constraint published_requires_category {
  on: [create, update]
  when: self.status == "published"
  condition: self.category != nil
  error: "Published posts must have a category"
}

@constraint unique_slug_per_status {
  on: [create, update]
  condition: !Post.exists?(
    slug: self.slug,
    status: ["published", "scheduled"],
    id_not: self.id
  )
  error: "A post with this slug already exists"
}
```

### Runtime Invariants

```
@invariant metrics_non_negative {
  condition:
    self.metrics.view_count >= 0 &&
    self.metrics.comment_count >= 0 &&
    self.metrics.like_count >= 0
  error: "Metrics cannot be negative"
}

@invariant price_positive {
  condition: self.pricing.regular > 0.0
  error: "Price must be positive"
}
```

---

## Error Handling

### Rescue Blocks

```
// Basic rescue
operation() rescue |err| {
  Logger.error("Operation failed", error: err)
}

// Rescue with fallback
let result = risky_operation() rescue |err| {
  Logger.warn("Fallback used", error: err)
  return default_value
}

// Multiple operations
@async {
  Email.send(user, "welcome") rescue |err| {
    Logger.error("Email failed", error: err)
  }

  Analytics.track(event) rescue |err| {
    Logger.warn("Analytics failed", error: err)
  }
}
```

### Unwrap Operator

```
// ! unwraps or panics (use for invariants)
let user = Context.current_user!()  // Panics if nil
let value = self.required_field!    // Panics if nil

// Use when you KNOW it can't be nil
@after create @transaction {
  let user_id = Context.current_user!().id
  self.created_by = user_id
}
```

---

## Query Language

### Basic Queries

```
// Find by ID
Post.find(id)

// Find by attribute
Post.find_by(slug: "hello-world")

// Where conditions
Post.where(status: "published")
Post.where(status: "published", featured: true)

// Comparison operators
Post.where(view_count > 1000)
Post.where(created_at >= Time.now() - 7.days)

// Array membership
Post.where(status in ["published", "featured"])
Post.where(status not_in ["draft", "archived"])
```

### Query Scopes

```
// Define reusable scopes
@scope published {
  where: { status: "published", published_at: { lte: Time.now() } }
  order_by: "published_at DESC"
}

@scope featured {
  where: { featured: true, status: "published" }
  order_by: "published_at DESC"
}

@scope by_category(category_id: uuid) {
  where: { category_id: category_id, status: "published" }
}

@scope search(query: string) {
  where: {
    or: [
      { title: { ilike: "%#{query}%" } },
      { content: { ilike: "%#{query}%" } }
    ],
    status: "published"
  }
}

// Use scopes
Post.published
Post.featured
Post.by_category(category_id)
Post.search("rails")

// Chain scopes
Post.published.by_category(id).order_by("created_at DESC")
```

### Aggregations

```
// Count
count(Post.all)
count(Post.where(status: "published"))

// Sum
sum(Order.all, :total)
sum(Order.where(status: "completed"), :total)

// Average
avg(Product.all, :price)

// Min/Max
min(Product.all, :price)
max(Product.all, :price)
```

### Advanced Queries

```
// Joins
Post.joins(:author).where(authors: { role: "admin" })

// Includes (eager loading)
Post.includes(:author, :category, :comments)

// Order
Post.order_by("created_at DESC")
Post.order_by("view_count DESC", "created_at DESC")

// Limit/Offset
Post.limit(10)
Post.offset(20).limit(10)

// Pluck (select specific fields)
Post.pluck(:id, :title)

// Exists
Post.exists?(slug: "hello-world")
```

---

## Complete Examples

### Example 1: Blog Post Resource

```
@strict nullability

/// Blog post with content, categorization, and publishing workflow
resource Post {
  id: uuid! @primary @auto

  // Content
  title: string! @min(5) @max(200)
  slug: string! @unique
  excerpt: text? @max(500)
  content: text! @min(100)

  // Author & Categorization
  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  category: Category? {
    foreign_key: "category_id"
    on_delete: set_null
  }

  tags: array<uuid>! @default([])

  // SEO
  seo: {
    meta_title: string? @max(60)
    meta_description: string? @max(160)
    og_image: url?
  }? @default({})

  // Status & Visibility
  status: enum ["draft", "published", "archived", "scheduled"]! @default("draft")
  visibility: enum ["public", "private", "password_protected"]! @default("public")
  password_hash: string?

  // Scheduling
  published_at: timestamp?
  scheduled_for: timestamp?

  // Metrics
  metrics: {
    view_count: int! @default(0)
    comment_count: int! @default(0)
    like_count: int! @default(0)
  }! @default({})

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  // Relationships
  @has_many Comment as "comments" {
    foreign_key: "post_id"
    on_delete: cascade
    order_by: "created_at DESC"
  }

  // Custom functions
  @function generate_excerpt(content: text, max_length: int) -> string {
    let stripped = HTML.strip_tags(content)
    let truncated = String.truncate(stripped, max_length)
    return truncated + "..."
  }

  // Lifecycle
  @before create @transaction {
    self.slug = String.slugify(self.title)
    self.reading_time = Text.calculate_reading_time(self.content, words_per_minute: 200)

    if self.excerpt == nil {
      self.excerpt = generate_excerpt(self.content, 200)
    }

    if self.visibility == "password_protected" && self.password_hash != nil {
      self.password_hash = Crypto.hash(self.password_hash!, algorithm: "bcrypt")
    }
  }

  @after create @transaction {
    if self.status == "published" && self.published_at == nil {
      self.published_at = Time.now()
    }

    @async {
      Notify.send(
        to: self.author.subscribers,
        template: "new_post",
        data: { post_title: self.title, post_url: self.url }
      ) rescue |err| {
        Logger.error("Failed to notify subscribers", error: err)
      }
    }
  }

  @after update @transaction {
    if self.content_changed? {
      Revision.create!({
        post_id: self.id,
        content: self.previous_value(:content),
        created_by: Context.current_user!().id
      })
    }

    if self.status_changed_to?("published") {
      self.published_at = Time.now()

      @async {
        SearchIndex.update(self) rescue |err| {
          Logger.warn("Search index update failed", error: err)
        }
      }
    }
  }

  // Constraints
  @constraint published_requires_category {
    on: [create, update]
    when: self.status == "published"
    condition: self.category != nil
    error: "Published posts must have a category"
  }

  @constraint password_protected_requires_password {
    on: [create, update]
    when: self.visibility == "password_protected"
    condition: self.password_hash != nil
    error: "Password-protected posts require a password"
  }

  @constraint scheduled_requires_future_date {
    on: [create, update]
    when: self.status == "scheduled"
    condition: self.scheduled_for != nil && self.scheduled_for! > Time.now()
    error: "Scheduled posts require a future date"
  }

  // Invariants
  @invariant metrics_non_negative {
    condition:
      self.metrics.view_count >= 0 &&
      self.metrics.comment_count >= 0 &&
      self.metrics.like_count >= 0
    error: "Metrics cannot be negative"
  }

  // Computed
  @computed is_published: bool! {
    return self.status == "published" &&
           self.published_at != nil &&
           self.published_at! <= Time.now()
  }

  @computed url: string! {
    return "/blog/" + self.slug
  }

  @computed can_comment: bool! {
    return self.allow_comments && self.is_published
  }

  // Middleware
  @on list: [cache(300), filter_by_status]
  @on get: [increment_metric(:view_count), cache(600)]
  @on create: [auth, rate_limit(5, per: "hour")]
  @on update: [auth, author_or_editor]
  @on delete: [auth, author_or_admin]

  // Scopes
  @scope published {
    where: { status: "published", published_at: { lte: Time.now() } }
    order_by: "published_at DESC"
  }

  @scope search(query: string) {
    where: {
      or: [
        { title: { ilike: "%#{query}%" } },
        { content: { ilike: "%#{query}%" } }
      ],
      status: "published"
    }
  }
}
```

### Example 2: E-commerce Product

```
@strict nullability

/// Product catalog item with pricing, inventory, and categorization
resource Product {
  id: uuid! @primary @auto

  // Basic info
  name: string! @min(3) @max(200)
  slug: string! @unique
  description: text!

  // Pricing (grouped)
  pricing: {
    regular: float! @min(0.0)
    sale: float? @min(0.0)
    cost: float! @min(0.0)
    currency: enum ["USD", "EUR", "GBP"]! @default("USD")
  }!

  // Inventory (grouped)
  inventory: {
    sku: string! @unique
    quantity: int! @default(0) @min(0)
    low_threshold: int! @default(10)
  }!

  // Categorization
  category: Category! {
    foreign_key: "category_id"
    on_delete: restrict
  }

  brand: Brand? {
    foreign_key: "brand_id"
    on_delete: set_null
  }

  tag_ids: array<uuid>! @default([])

  // Status
  status: enum ["draft", "active", "archived"]! @default("draft")
  featured: bool! @default(false)

  // Media
  images: array<{
    url: url!
    alt: string?
    order: int! @default(0)
  }>! @default([])

  // Metrics
  metrics: {
    view_count: int! @default(0)
    order_count: int! @default(0)
    revenue_total: float! @default(0.0)
  }! @default({})

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  // Relationships
  @has_many Review as "reviews" {
    foreign_key: "product_id"
    on_delete: cascade
  }

  // Lifecycle
  @before create {
    self.slug = String.slugify(self.name)
  }

  @after update @transaction {
    if self.inventory.quantity <= self.inventory.low_threshold {
      @async {
        Notify.send(
          to: admin_users,
          template: "low_stock_alert",
          data: {
            product_name: self.name,
            sku: self.inventory.sku,
            quantity: self.inventory.quantity
          }
        ) rescue |err| {
          Logger.error("Low stock alert failed", error: err)
        }
      }
    }
  }

  // Constraints
  @constraint sale_price_less_than_regular {
    on: [create, update]
    when: self.pricing.sale != nil
    condition: self.pricing.sale! < self.pricing.regular
    error: "Sale price must be less than regular price"
  }

  @constraint active_requires_stock {
    on: [create, update]
    when: self.status == "active"
    condition: self.inventory.quantity > 0
    error: "Cannot activate product with zero stock"
  }

  // Computed
  @computed on_sale: bool! {
    return self.pricing.sale != nil && self.pricing.sale! < self.pricing.regular
  }

  @computed in_stock: bool! {
    return self.inventory.quantity > 0
  }

  @computed effective_price: float! {
    return self.pricing.sale ?? self.pricing.regular
  }

  @computed formatted_price: string! {
    let price = self.effective_price
    let symbol = match self.pricing.currency {
      "USD" => "$",
      "EUR" => "€",
      "GBP" => "£"
    }
    return "#{symbol}#{Number.format(price, decimals: 2)}"
  }

  // Middleware
  @on list: [cache(300)]
  @on get: [increment_metric(:view_count), cache(600)]
  @on create: [auth, admin]
  @on update: [auth, admin]
  @on delete: [auth, admin, check_no_orders]

  // Scopes
  @scope active {
    where: { status: "active", inventory: { quantity: { gt: 0 } } }
  }

  @scope on_sale {
    where: { pricing: { sale: { not_null: true } } }
  }
}
```

---

## Summary

**Key Features:**

1. ✅ **Explicit Nullability** - `!` (required) vs `?` (optional)
2. ✅ **Namespaced Stdlib** - `String.slugify()` vs `slugify()`
3. ✅ **Structural Types** - `array<T>`, `hash<K,V>`, inline structs
4. ✅ **Transaction Boundaries** - `@transaction`, `@async`
5. ✅ **Error Handling** - `rescue |err| { }` blocks
6. ✅ **Custom Functions** - `@function name() -> type { }`
7. ✅ **Declarative Constraints** - Named, testable validations
8. ✅ **Runtime Invariants** - `@invariant` for assertions
9. ✅ **Query Scopes** - Reusable query patterns
10. ✅ **Relationship Metadata** - `on_delete`, `foreign_key` explicit

**Progressive Disclosure:**
- Simple: 3 lines is a valid resource
- Medium: Add validations, hooks
- Advanced: Full transaction control, error handling, complex constraints

**For LLMs:**
- Zero ambiguity about types and nullability
- Clear provenance of every function
- Structured patterns to replicate
- Error handling is built into syntax
- Can validate logic without executing

**For Humans:**
- Intention is crystal clear
- Less cognitive load
- Safer (constraints prevent bad states)
- More maintainable (everything explicit)

---

**Status:** Ready for implementation
**Next:** Build parser and begin compiler implementation
