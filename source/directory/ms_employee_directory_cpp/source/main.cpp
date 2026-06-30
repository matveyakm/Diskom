#include <crow.h>
#include <simdjson.h>
#include <sqlite3.h>
#include <iostream>
#include <vector>
#include <string>
#include <memory>


struct db_stmts {
  sqlite3_stmt *get_employee_dir_stmt = nullptr;
  sqlite3_stmt *add_employee_dir_stmt = nullptr;
  sqlite3_stmt *delete_employee_dir_stmt = nullptr;
  sqlite3_stmt *get_employees_dir_stmt = nullptr;
  sqlite3_stmt* update_employee_dir_stmt = nullptr;
void finalize()
{
   sqlite3_finalize(get_employee_dir_stmt);
   sqlite3_finalize(delete_employee_dir_stmt);
   sqlite3_finalize(get_employees_dir_stmt);
   sqlite3_finalize(add_employee_dir_stmt);
   sqlite3_finalize(update_employee_dir_stmt);
}
};

crow::response get_employee_dir_handle(sqlite3 *db, sqlite3_stmt *stmt, int64_t id);
crow::response get_employees_dir_handle(sqlite3 *db, sqlite3_stmt *stmt);
crow::response add_employee_dir_handle(const crow::request &req, sqlite3 *db, const db_stmts& stmts);
crow::response delete_employee_dir_handle(sqlite3 *db, sqlite3_stmt *stmt,
                                          int64_t id);
crow::response update_employee_dir_handle(const crow::request &req, sqlite3 *db,
                                          int64_t id, const db_stmts& stmts);

struct Employee_Directory {
  int64_t Id = 0;
  int64_t Basis = 0;
  int64_t Status = 0;
  int64_t Prefix = 0;
  std::string Name = "";
  std::string Description = "";
  std::string IdTransaction = "";

  std::string Serialize() {
  return {
      std::format(
          R"({{"id":{},"name":"{}","description":"{}","status":{},"basis":{},"prefix":{},"id_transaction":"{}"}})",
          Id, Name, Description, Status, Basis, Prefix, IdTransaction)
  };
    }
};

int main() {
  db_stmts stmts;
  crow::SimpleApp server;
  sqlite3 *db = nullptr;

  if (sqlite3_open("employee_directory.db", &db) != SQLITE_OK) {
    std::cerr << "failed to open DB";
    stmts.finalize();
    sqlite3_close(db);
    db = nullptr;
    return SQLITE_CANTOPEN;
  }

  sqlite3_prepare_v2(db, "select * from employee_directory where id = ?", -1,
                     &stmts.get_employee_dir_stmt, nullptr);

  sqlite3_prepare_v2(db,
      "INSERT INTO employee_directory (id, name, description, status, basis, prefix, id_transaction) "
      "VALUES (:id, :name, :description, :status, :basis, :prefix, :id_transaction)",
      -1, &stmts.add_employee_dir_stmt, nullptr);

  sqlite3_prepare_v2(db, "delete from employee_directory where id = ?", -1,
                     &stmts.delete_employee_dir_stmt, nullptr);

  sqlite3_prepare_v2(db, "select * from employee_directory", -1,
                     &stmts.get_employees_dir_stmt, nullptr);

  sqlite3_prepare_v2(db,
      "UPDATE employee_directory SET "
      "name = COALESCE(:name, name), "
      "description = COALESCE(:description, description), "
      "status = COALESCE(:status, status), "
      "basis = COALESCE(:basis, basis), "
      "prefix = COALESCE(:prefix, prefix), "
      "id_transaction = COALESCE(:id_transaction, id_transaction) "
      "WHERE id = :id",
      -1, &stmts.update_employee_dir_stmt, nullptr);

  CROW_ROUTE(server, "/employee_directory_cpp")
      .methods(
          crow::HTTPMethod::POST, crow::HTTPMethod::GET,
          crow::HTTPMethod::DELETE, crow::HTTPMethod::PUT
      )([&db, &stmts](const crow::request &req) {
        int64_t id = 0;
        std::cout << "start switch\n";
        switch (req.method) {
        case crow::HTTPMethod::POST:
          return add_employee_dir_handle(req, db, stmts);
          break;

        case crow::HTTPMethod::GET:
          char *params;
          params = req.url_params.get("id");
          if (params) {
            id = std::stol(req.url_params.get("id"));
            return get_employee_dir_handle(db, stmts.get_employee_dir_stmt, id);
          }
          std::cout << "request Get all\n";
          return get_employees_dir_handle(db, stmts.get_employees_dir_stmt);

          break;

        case crow::HTTPMethod::DELETE:
          id = std::stol(req.url_params.get("id"));
          return delete_employee_dir_handle(db, stmts.delete_employee_dir_stmt,
                                            id);
          break;
        case crow::HTTPMethod::PUT:
            id = std::stol(req.url_params.get("id"));
            return update_employee_dir_handle(req, db, id,stmts);
            break;
        default:
          return crow::response(500, "METHOD NOT IMPLEMENTED");
          break;
        }
      });

  server.bindaddr("172.25.0.62").port(8443).multithreaded().run();
  stmts.finalize();
  sqlite3_close(db);
  db = nullptr;
}

crow::response get_employee_dir_handle(sqlite3 *db, sqlite3_stmt *stmt,
                                       int64_t id) {
  sqlite3_reset(stmt);
  sqlite3_bind_int64(stmt, 1, id);

  int rc = sqlite3_step(stmt);
  if (rc == SQLITE_DONE) {
      return crow::response{404, R"({"error":"not found"})"};
  }
  if (rc != SQLITE_ROW) {
    std::println(stderr, "Execution crashed: {}", sqlite3_errmsg(db));
    return crow::response{500, "Internal Server Error"};
  }
  Employee_Directory directory{};
  directory.Id = sqlite3_column_int(stmt, 0);
  directory.Name = (const char *)sqlite3_column_text(stmt, 1);
  directory.Description = (const char *)sqlite3_column_text(stmt, 2);
  directory.Status = sqlite3_column_int(stmt, 3);
  directory.Basis = sqlite3_column_int64(stmt, 4);
  directory.Prefix = sqlite3_column_int(stmt, 5);
  if (auto t = sqlite3_column_text(stmt, 6))
    directory.IdTransaction = (const char *)t;

  return crow::response{200, directory.Serialize()};
}

crow::response add_employee_dir_handle(const crow::request &req, sqlite3 *db, const db_stmts& stmts) {
  sqlite3_stmt* stmt = stmts.add_employee_dir_stmt;
  sqlite3_reset(stmt);

  simdjson::ondemand::parser parser;
  simdjson::padded_string body(req.body);
  simdjson::ondemand::document doc = parser.iterate(body);

  Employee_Directory directory;

  if(auto result = doc.find_field_unordered("id"); result.error() == simdjson::SUCCESS)
  {
      directory.Id = result.value().get_int64().value();
      sqlite3_bind_int64(stmt, sqlite3_bind_parameter_index(stmt, ":id"), directory.Id);
  }
  if(auto result = doc.find_field_unordered("name"); result.error() == simdjson::SUCCESS)
  {
      directory.Name = result.value().get_string().value();
      sqlite3_bind_text(stmt, sqlite3_bind_parameter_index(stmt, ":name"), directory.Name.data(), (int)directory.Name.size(), SQLITE_TRANSIENT );
  }
  if(auto result = doc.find_field_unordered("description"); result.error() == simdjson::SUCCESS)
  {
      directory.Description = result.value().get_string().value();
      sqlite3_bind_text(stmt, sqlite3_bind_parameter_index(stmt, ":description"), directory.Description.data(), (int)directory.Description.size(), SQLITE_TRANSIENT );
  }
  if(auto result = doc.find_field_unordered("status"); result.error() == simdjson::SUCCESS)
  {
      directory.Status = result.value().get_int32().value();
      sqlite3_bind_int64(stmt, sqlite3_bind_parameter_index(stmt, ":status"), directory.Status);
  }
  if(auto result = doc.find_field_unordered("basis"); result.error() == simdjson::SUCCESS)
  {
      directory.Basis = result.value().get_int32().value();
      sqlite3_bind_int64(stmt, sqlite3_bind_parameter_index(stmt, ":basis"), directory.Basis);
  }
  if(auto result = doc.find_field_unordered("prefix"); result.error() == simdjson::SUCCESS)
  {
      directory.Prefix = result.value().get_int32().value();
      sqlite3_bind_int64(stmt, sqlite3_bind_parameter_index(stmt, ":prefix"), directory.Prefix);
  }
  if(auto result = doc.find_field_unordered("id_transaction"); result.error() == simdjson::SUCCESS)
  {
      directory.IdTransaction = result.value().get_string().value();
      sqlite3_bind_text(stmt, sqlite3_bind_parameter_index(stmt, ":id_transaction"), directory.IdTransaction.data(), (int)directory.IdTransaction.size(), SQLITE_TRANSIENT);
  }

  int rc = sqlite3_step(stmt);
  if (rc != SQLITE_DONE) {
      return crow::response{500, sqlite3_errmsg(db)};
  }

  if(!directory.Id) [[likely]]
  {
      directory.Id = sqlite3_last_insert_rowid(db);
  }

  return get_employee_dir_handle(db, stmts.get_employee_dir_stmt, directory.Id);
}

crow::response delete_employee_dir_handle(sqlite3 *db, sqlite3_stmt *stmt,
                                          int64_t id) {
  sqlite3_reset(stmt);
  sqlite3_bind_int64(stmt, 1, id);

  int rc = sqlite3_step(stmt);
  if (rc == SQLITE_DONE) {
    return crow::response{
        200,
        R"({"status": "success","message": "resource has been deleted"})"};
  }
  std::println(stderr, "Execution crashed: {}", sqlite3_errmsg(db));
  return crow::response{500, "Internal Server Error"};
}

crow::response get_employees_dir_handle(sqlite3 *db, sqlite3_stmt *stmt) {
    sqlite3_reset(stmt);

    int rc = sqlite3_step(stmt);
    if (rc == SQLITE_DONE) {
      return crow::response{404, R"({"error":"not found"})"};
    }
    if (rc != SQLITE_ROW) {
      std::println(stderr, "Execution crashed: {}", sqlite3_errmsg(db));
      return crow::response{500, "Internal Server Error"};
    }
    std::vector<Employee_Directory> directories(10);
    std::string result = "";

    for (int i = 0; rc == SQLITE_ROW; i++) {
      directories[i].Id = sqlite3_column_int64(stmt, 0);
      directories[i].Name = (const char *)sqlite3_column_text(stmt, 1);
      directories[i].Description = (const char *)sqlite3_column_text(stmt, 2);
      directories[i].Status = sqlite3_column_int64(stmt, 3);
      directories[i].Basis = sqlite3_column_int64(stmt, 4);
      directories[i].Prefix = sqlite3_column_int64(stmt, 5);
      if (auto t = sqlite3_column_text(stmt, 6))
        directories[i].IdTransaction = (const char *)t;
      if (!result.empty())
        result += ',';

      result += directories[i].Serialize();
      rc = sqlite3_step(stmt);
    }
    result = std::format(R"({{"data":[{}]}})", result);

    return crow::response{200, result};
}

crow::response update_employee_dir_handle(const crow::request& req, sqlite3* db, int64_t id, const db_stmts& stmts) {
    sqlite3_stmt* update_stmt = stmts.update_employee_dir_stmt;
    sqlite3_stmt* get_stmt = stmts.get_employee_dir_stmt;

    sqlite3_reset(update_stmt);

    simdjson::ondemand::parser parser;
    simdjson::padded_string body(req.body);
    simdjson::ondemand::document doc = parser.iterate(body);

    Employee_Directory directory;

    sqlite3_bind_int64(update_stmt, sqlite3_bind_parameter_index(update_stmt, ":id"), id);

    if (auto result = doc.find_field_unordered("name"); result.error() == simdjson::SUCCESS)
    {
        directory.Name = result.value().get_string().value();
        sqlite3_bind_text(update_stmt, sqlite3_bind_parameter_index(update_stmt, ":name"),
                          directory.Name.data(), (int)directory.Name.size(), SQLITE_TRANSIENT);
    }
    if (auto result = doc.find_field_unordered("description"); result.error() == simdjson::SUCCESS)
    {
        directory.Description = result.value().get_string().value();
        sqlite3_bind_text(update_stmt, sqlite3_bind_parameter_index(update_stmt, ":description"),
                          directory.Description.data(), (int)directory.Description.size(), SQLITE_TRANSIENT);
    }
    if (auto result = doc.find_field_unordered("status"); result.error() == simdjson::SUCCESS)
    {
        directory.Status = result.value().get_int32().value();
        sqlite3_bind_int64(update_stmt, sqlite3_bind_parameter_index(update_stmt, ":status"), directory.Status);
    }
    if (auto result = doc.find_field_unordered("basis"); result.error() == simdjson::SUCCESS)
    {
        directory.Basis = result.value().get_int32().value();
        sqlite3_bind_int64(update_stmt, sqlite3_bind_parameter_index(update_stmt, ":basis"), directory.Basis);
    }
    if (auto result = doc.find_field_unordered("prefix"); result.error() == simdjson::SUCCESS)
    {
        directory.Prefix = result.value().get_int32().value();
        sqlite3_bind_int64(update_stmt, sqlite3_bind_parameter_index(update_stmt, ":prefix"), directory.Prefix);
    }
    if (auto result = doc.find_field_unordered("id_transaction"); result.error() == simdjson::SUCCESS)
    {
        directory.IdTransaction = result.value().get_string().value();
        sqlite3_bind_text(update_stmt, sqlite3_bind_parameter_index(update_stmt, ":id_transaction"),
                          directory.IdTransaction.data(), (int)directory.IdTransaction.size(), SQLITE_TRANSIENT);
    }

    int rc = sqlite3_step(update_stmt);
    if (rc != SQLITE_DONE) {
        return crow::response{500, sqlite3_errmsg(db)};
    }
    if (sqlite3_changes(db) == 0) {
        return crow::response{404, "No record found with the given id"};
    }

    return get_employee_dir_handle(db, get_stmt, id);
}
