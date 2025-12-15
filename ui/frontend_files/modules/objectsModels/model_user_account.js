/**
 * JavaScript equivalent of Go structs from model_user_account.go
 */

// Constants matching Go enums
export const UserRoles = {
  USER: 0,  // UserRole
  ADMIN: 1  // AdminRole
};

export const USER_ACCOUNT_DATA_COLL = "webconsoleData.snapshots.userAccountData";

export class DBUserAccount {
  constructor() {
    this.username = "";
    this.hashedPassword = "";
    this.role = UserRoles.USER;
  }
}

export class CreateUserAccountParams {
  constructor() {
    this.username = "";
    this.password = "";
  }
}

export class ChangePasswordParams {
  constructor() {
    this.password = "";
  }
}

export class GetUserAccountResponse {
  constructor() {
    this.username = "";
    this.role = UserRoles.USER;
  }
}

// Note: Password hashing methods omitted as they would be handled server-side in Go
