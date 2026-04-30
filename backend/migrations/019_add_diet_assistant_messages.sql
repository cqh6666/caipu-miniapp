CREATE TABLE IF NOT EXISTS diet_assistant_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  kitchen_id INTEGER NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (kitchen_id) REFERENCES kitchens(id)
);

CREATE INDEX IF NOT EXISTS idx_diet_assistant_messages_user_kitchen_id
  ON diet_assistant_messages(user_id, kitchen_id, id);

CREATE INDEX IF NOT EXISTS idx_diet_assistant_messages_created_at
  ON diet_assistant_messages(created_at);
