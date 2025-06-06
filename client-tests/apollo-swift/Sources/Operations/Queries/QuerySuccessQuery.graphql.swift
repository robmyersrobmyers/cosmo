// @generated
// This file was automatically generated and should not be edited.

@_exported import ApolloAPI

public class QuerySuccessQuery: GraphQLQuery {
  public static let operationName: String = "QuerySuccess"
  public static let operationDocument: ApolloAPI.OperationDocument = .init(
    definition: .init(
      #"query QuerySuccess { employees { __typename id isAvailable } }"#
    ))

  public init() {}

  public struct Data: ApolloSwift.SelectionSet {
    public let __data: DataDict
    public init(_dataDict: DataDict) { __data = _dataDict }

    public static var __parentType: any ApolloAPI.ParentType { ApolloSwift.Objects.Query }
    public static var __selections: [ApolloAPI.Selection] { [
      .field("employees", [Employee?]?.self),
    ] }

    public var employees: [Employee?]? { __data["employees"] }

    /// Employee
    ///
    /// Parent Type: `Employee`
    public struct Employee: ApolloSwift.SelectionSet {
      public let __data: DataDict
      public init(_dataDict: DataDict) { __data = _dataDict }

      public static var __parentType: any ApolloAPI.ParentType { ApolloSwift.Objects.Employee }
      public static var __selections: [ApolloAPI.Selection] { [
        .field("__typename", String.self),
        .field("id", Int.self),
        .field("isAvailable", Bool.self),
      ] }

      public var id: Int { __data["id"] }
      public var isAvailable: Bool { __data["isAvailable"] }
    }
  }
}
