# ETL Configuration

## Sources
- **pizza_db** (sqlite3) → `~/apps/projects/etlctl/etls/data/pizza.db`
- **pizza_2_db** (sqlite3) → `~/apps/projects/etlctl/etls/data/pizza_2.db`

## Queries
- **get_pizzas**
  ```sql
  SELECT name, toppings FROM pizza;

  ```

## Targets
- **delivery_db** (sqlite3) → `~/apps/projects/etlctl/etls/data/delivery.db`

## Pipelines

### **Pipeline 1**
- **Sources:** pizza_db, pizza_2_db
- **Query:** `get_pizzas`
- **Target:** `delivery_db.delivery`
- **Field Mappings:**
  - `name` → `name`
  - `toppings` → `toppings`
