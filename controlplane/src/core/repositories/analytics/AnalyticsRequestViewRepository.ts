import { PlainMessage } from '@bufbuild/protobuf';
import {
  AnalyticsConfig,
  AnalyticsViewFilterOperator,
  AnalyticsViewGroupName,
  AnalyticsViewResult,
  AnalyticsViewResultFilter,
  AnalyticsViewRow,
  AnalyticsViewRowValue,
  Unit,
} from '@wundergraph/cosmo-connect/dist/platform/v1/platform_pb';
import { ClickHouseClient } from '../../clickhouse/index.js';
import {
  BaseFilters,
  ColumnMetaData,
  buildAnalyticsViewColumns,
  buildAnalyticsViewFilters,
  buildCoercedFilterSqlStatement,
  buildColumnsFromNames,
  coerceFilterValues,
  fillColumnMetaData,
} from './util.js';

/**
 * Repository for clickhouse analytics data
 */
export class AnalyticsRequestViewRepository {
  constructor(private client: ClickHouseClient) {}

  public columnMetadata: ColumnMetaData = {
    durationInNano: {
      name: 'durationInNano',
      unit: Unit.Nanoseconds,
      type: 'number',
      title: 'Duration',
    },
    unixTimestamp: {
      name: 'unixTimestamp',
      unit: Unit.UnixTimestamp,
      type: 'number',
      title: 'Timestamp',
    },
    statusCode: {
      unit: Unit.StatusCode,
      title: 'Status Code',
      isHidden: true,
    },
    statusMessage: {
      title: 'Status Message',
      isHidden: true,
    },
    operationHash: {
      isHidden: true,
      title: 'Operation Hash',
    },
    operationName: {
      title: 'Operation Name',
    },
    operationType: {
      title: 'Operation Type',
    },
    operationContent: {
      title: 'Operation Content',
      isCta: true,
      unit: Unit.CodeBlock,
    },
    httpStatusCode: {
      title: 'HTTP Status Code',
    },
    httpHost: {
      isHidden: true,
      title: 'HTTP Host',
    },
    httpUserAgent: {
      title: 'User Agent',
      isCta: true,
    },
    httpMethod: {
      isHidden: true,
      title: 'HTTP Method',
    },
    httpTarget: {
      isHidden: true,
      title: 'HTTP Target',
    },
    traceId: {
      unit: Unit.TraceID,
      title: 'Trace ID',
    },
    totalRequests: {
      type: 'number',
      title: 'Total Requests',
    },
    p95: {
      type: 'number',
      unit: Unit.Nanoseconds,
      title: 'P95 Latency',
    },
    errorsWithRate: {
      title: 'Errors (Rate%)',
    },
    rate: {
      title: 'Rate',
    },
    lastCalled: {
      unit: Unit.UnixTimestamp,
      type: 'number',
      title: 'Last Called',
    },
    clientName: {
      title: 'Client Name',
    },
    clientVersion: {
      title: 'Client Version',
    },
  };

  public baseFilters: BaseFilters = {
    operationName: {
      dbField: "SpanAttributes [ 'wg.operation.name' ]",
      dbClause: 'where',
      columnName: 'operationName',
      title: 'Operation Name',
      options: [],
    },
    operationType: {
      dbField: "SpanAttributes [ 'wg.operation.type' ]",
      dbClause: 'where',
      title: 'Operation Type',
      columnName: 'operationType',
      options: [
        {
          operator: AnalyticsViewFilterOperator.EQUALS,
          label: 'Query',
          value: 'query',
        },
        {
          operator: AnalyticsViewFilterOperator.EQUALS,
          label: 'Mutation',
          value: 'mutation',
        },
        {
          operator: AnalyticsViewFilterOperator.EQUALS,
          label: 'Subscription',
          value: 'subscription',
        },
      ],
    },
    durationInNano: {
      dbField: 'Duration',
      dbClause: 'where',
      title: 'Duration',
      columnName: 'durationInNano',
      options: [
        {
          operator: AnalyticsViewFilterOperator.GREATER_THAN,
          label: '> 1s',
          value: '1000000000',
        },
        {
          operator: AnalyticsViewFilterOperator.LESS_THAN,
          label: '< 1s',
          value: '1000000000',
        },
      ],
    },
    p95: {
      dbField: 'quantile(0.95)(Duration)',
      dbClause: 'having',
      title: 'P95 Latency',
      columnName: 'p95',
      options: [
        {
          operator: AnalyticsViewFilterOperator.LESS_THAN,
          label: '< 150ms',
          value: '150000000',
        },
        {
          operator: AnalyticsViewFilterOperator.LESS_THAN,
          label: '< 1000ms',
          value: '1000000000',
        },
        {
          operator: AnalyticsViewFilterOperator.LESS_THAN,
          label: '< 2000ms',
          value: '2000000000',
        },
        {
          operator: AnalyticsViewFilterOperator.GREATER_THAN_OR_EQUAL,
          label: '>= 2000ms',
          value: '2000000000',
        },
      ],
    },
    clientName: {
      dbField: "SpanAttributes [ 'wg.client.name' ]",
      dbClause: 'where',
      columnName: 'clientName',
      title: 'Client Name',
      options: [],
    },
    httpStatusCode: {
      dbField: "SpanAttributes [ 'http.status_code' ]",
      dbClause: 'where',
      columnName: 'httpStatusCode',
      title: 'Http Status Code',
      options: [],
    },
  };

  private getViewData(
    name: AnalyticsViewGroupName,
    baseWhereSql: string,
    baseHavingSql: string,
    basePaginationSql: string,
    queryParams: Record<string, string | number>,
  ) {
    let query = ``;

    switch (name) {
      case AnalyticsViewGroupName.None: {
        query = `
          SELECT
              TraceId as traceId,
              toString(toUnixTimestamp(Timestamp)) as unixTimestamp,
              -- DateTime64 is returned as a string
              SpanAttributes['wg.operation.name'] as operationName,
              SpanAttributes [ 'wg.operation.type' ] as operationType,
              Duration as durationInNano,
              StatusCode as statusCode,
              StatusMessage as statusMessage,
              SpanAttributes [ 'wg.operation.hash' ] as operationHash,
              SpanAttributes [ 'wg.operation.content' ] as operationContent,
              SpanAttributes [ 'http.status_code' ] as httpStatusCode,
              SpanAttributes [ 'http.host' ] as httpHost,
              SpanAttributes [ 'http.user_agent' ] as httpUserAgent,
              SpanAttributes [ 'http.method' ] as httpMethod,
              SpanAttributes [ 'http.target' ] as httpTarget,
              SpanAttributes [ 'wg.client.name' ] as clientName
          FROM
              ${this.client.database}.otel_traces
          WHERE
            -- Only root spans
            (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
            ${baseWhereSql}
          ORDER BY
              Timestamp DESC 
            ${basePaginationSql}
        `;
        break;
      }
      case AnalyticsViewGroupName.OperationName: {
        query = `
          SELECT
            SpanAttributes['wg.operation.name'] as operationName,
            SpanAttributes [ 'wg.operation.type' ] as operationType,
            COUNT(*) as totalRequests,
            quantile(0.95)(Duration) as p95,
            CONCAT(
                toString(SUM(if(StatusCode = 'STATUS_CODE_ERROR' OR position(SpanAttributes['http.status_code'],'4') = 1, 1, 0))), 
                ' (',
                toString(round(SUM(if(StatusCode = 'STATUS_CODE_ERROR' OR position(SpanAttributes['http.status_code'],'4') = 1, 1, 0)) / COUNT(*) * 100, 2)),
                '%)'
            ) as errorsWithRate,
            toString(toUnixTimestamp(MAX(Timestamp))) as lastCalled
          FROM
              ${this.client.database}.otel_traces
          WHERE
            (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
            ${baseWhereSql}
          GROUP BY
            operationName,
            operationType
          ${baseHavingSql}
          ORDER BY
              totalRequests DESC
            ${basePaginationSql}
        `;
        break;
      }
      case AnalyticsViewGroupName.Client: {
        query = `
          SELECT
            SpanAttributes [ 'wg.client.name' ] as clientName,
            SpanAttributes [ 'wg.client.version' ] as clientVersion,
            COUNT(*) as totalRequests,
            quantile(0.95)(Duration) as p95,
            CONCAT(
                toString(SUM(if(StatusCode = 'STATUS_CODE_ERROR' OR position(SpanAttributes['http.status_code'],'4') = 1, 1, 0))), 
                ' (',
                toString(round(SUM(if(StatusCode = 'STATUS_CODE_ERROR' OR position(SpanAttributes['http.status_code'],'4') = 1, 1, 0)) / COUNT(*) * 100, 2)),
                '%)'
            ) as errorsWithRate,
            toString(toUnixTimestamp(MAX(Timestamp))) as lastCalled
          FROM
              ${this.client.database}.otel_traces
          WHERE
            (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
            ${baseWhereSql}
          GROUP BY
            clientName,
            clientVersion
          ${baseHavingSql}
          ORDER BY
              totalRequests DESC
            ${basePaginationSql}
        `;
        break;
      }
      case AnalyticsViewGroupName.HttpStatusCode: {
        query = `
          SELECT
            SpanAttributes['http.status_code'] as httpStatusCode,
            COUNT(*) as totalRequests,
            quantile(0.95)(Duration) as p95,
            toString(toUnixTimestamp(MAX(Timestamp))) as lastCalled
          FROM
              ${this.client.database}.otel_traces
          WHERE
              (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
              ${baseWhereSql}
          GROUP BY
              httpStatusCode
          ${baseHavingSql}
          ORDER BY
              totalRequests DESC
          ${basePaginationSql}
        `;
        break;
      }
    }

    return this.client?.queryPromise(query, queryParams);
  }

  private async getTotalCount(
    name: AnalyticsViewGroupName,
    baseWhereSql: string,
    baseHavingSql: string,
    queryParams: Record<string, string | number>,
  ): Promise<number> {
    let totalCountQuery = ``;

    switch (name) {
      case AnalyticsViewGroupName.None: {
        totalCountQuery = `
          SELECT COUNT(*) as count FROM ${this.client.database}.otel_traces
          WHERE
            -- Only root spans
            (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
          ${baseWhereSql}
        `;
        break;
      }
      case AnalyticsViewGroupName.OperationName: {
        totalCountQuery = `
          SELECT COUNT(*) as count FROM (
            SELECT
              SpanAttributes [ 'wg.operation.name' ]
            FROM
                ${this.client.database}.otel_traces
            WHERE
              (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
              ${baseWhereSql}
            GROUP BY
              SpanAttributes [ 'wg.operation.name' ],
              SpanAttributes [ 'wg.operation.type' ]
            ${baseHavingSql}
          )
        `;
        break;
      }
      case AnalyticsViewGroupName.Client: {
        totalCountQuery = `
          SELECT COUNT(*) as count FROM (
            SELECT
              SpanAttributes [ 'wg.client.name' ]
            FROM
                ${this.client.database}.otel_traces
            WHERE
              (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
              ${baseWhereSql}
            GROUP BY
              SpanAttributes [ 'wg.client.name' ],
              SpanAttributes [ 'wg.client.version' ]
            ${baseHavingSql}
          )
        `;
        break;
      }
      case AnalyticsViewGroupName.HttpStatusCode: {
        totalCountQuery = `
          SELECT COUNT(*) as count FROM (
            SELECT
              SpanAttributes['http.status_code'] as httpStatusCode
            FROM
              ${this.client.database}.otel_traces
            WHERE
              (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
              ${baseWhereSql}
            GROUP BY
                httpStatusCode
            ${baseHavingSql}
          )
        `;
      }
    }

    const countResult = await this.client?.queryPromise(totalCountQuery, queryParams);

    if (Array.isArray(countResult) && countResult.length > 0) {
      return countResult[0].count;
    }

    return 0;
  }

  private async getAllOperationNames(federatedGraphId: string, shouldExecute: boolean): Promise<string[]> {
    if (!shouldExecute) {
      return [];
    }

    // We need to get all operation names for the operationName filter options
    const allOperationNamesQuery = `
      SELECT DISTINCT SpanAttributes['wg.operation.name'] as operationName
      FROM ${this.client.database}.otel_traces
      WHERE
        (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
        AND SpanAttributes['wg.federated_graph.id'] = '${federatedGraphId}'
      ORDER BY Timestamp DESC
      LIMIT 1000
    `;

    const operationNamesResult = await this.client?.queryPromise(allOperationNamesQuery);

    const allOperationNames: string[] = [];
    if (Array.isArray(operationNamesResult)) {
      allOperationNames.push(...operationNamesResult.map((o) => o.operationName));
    }

    return allOperationNames;
  }

  private async getAllClients(federatedGraphId: string, shouldExecute: boolean): Promise<string[]> {
    if (!shouldExecute) {
      return [];
    }

    const query = `
      SELECT DISTINCT SpanAttributes [ 'wg.client.name' ] as clientName
      FROM ${this.client.database}.otel_traces
      WHERE 
        (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
        AND SpanAttributes['wg.federated_graph.id'] = '${federatedGraphId}'
      LIMIT 100
    `;

    const result = await this.client?.queryPromise(query);

    const clientNames: string[] = [];
    if (Array.isArray(result)) {
      clientNames.push(...result.map((c) => c.clientName));
    }

    return clientNames;
  }

  private async getAllStatusMessages(federatedGraphId: string, shouldExecute: boolean): Promise<string[]> {
    if (!shouldExecute) {
      return [];
    }

    const query = `
      SELECT DISTINCT StatusMessage as statusMessage
      FROM ${this.client.database}.otel_traces
      WHERE 
        (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
        AND SpanAttributes['wg.federated_graph.id'] = '${federatedGraphId}'
      LIMIT 100
    `;

    const result = await this.client?.queryPromise(query);

    const statusMessages: string[] = [];
    if (Array.isArray(result)) {
      statusMessages.push(...result.map((s) => s.statusMessage));
    }

    return statusMessages;
  }

  private async getAllHttpStatusCodes(federatedGraphId: string, shouldExecute: boolean): Promise<string[]> {
    if (!shouldExecute) {
      return [];
    }

    const query = `
      SELECT DISTINCT SpanAttributes [ 'http.status_code' ] as httpStatusCode
      FROM ${this.client.database}.otel_traces
      WHERE 
        (SpanKind = 'SPAN_KIND_SERVER' OR empty(ParentSpanId))
        AND SpanAttributes['wg.federated_graph.id'] = '${federatedGraphId}'
      LIMIT 100
    `;

    const result = await this.client?.queryPromise(query);

    const httpStatusCodes: string[] = [];
    if (Array.isArray(result)) {
      httpStatusCodes.push(...result.map((s) => s.httpStatusCode));
    }

    return httpStatusCodes;
  }

  private getFilters(
    name: AnalyticsViewGroupName,
    operationNames: string[],
    clientNames: string[],
    httpStatusCodes: string[],
  ): Record<string, PlainMessage<AnalyticsViewResultFilter>> {
    const filters = { ...this.baseFilters };
    filters.operationName = {
      ...filters.operationName,
      options: [
        ...filters.operationName.options,
        ...operationNames.map((op) => ({
          operator: AnalyticsViewFilterOperator.EQUALS,
          label: op || '-',
          value: op,
        })),
      ],
    };

    filters.clientName = {
      ...filters.clientName,
      options: clientNames.map((c) => ({
        operator: AnalyticsViewFilterOperator.EQUALS,
        label: c || '-',
        value: c,
      })),
    };

    filters.httpStatusCode = {
      ...filters.httpStatusCode,
      options: httpStatusCodes.map((sc) => ({
        operator: AnalyticsViewFilterOperator.EQUALS,
        label: sc || '-',
        value: sc,
      })),
    };

    switch (name) {
      case AnalyticsViewGroupName.None: {
        const { p95, ...rest } = filters;
        return rest;
      }
      case AnalyticsViewGroupName.OperationName: {
        const { durationInNano, clientName, statusMessages, ...rest } = filters;
        return rest;
      }
      case AnalyticsViewGroupName.Client: {
        const { clientName, p95 } = filters;
        return { clientName, p95 };
      }
      case AnalyticsViewGroupName.HttpStatusCode: {
        const { p95, httpStatusCode } = filters;
        return { p95, httpStatusCode };
      }
    }
  }

  public async getView(
    organizationId: string,
    federatedGraphId: string,
    name: AnalyticsViewGroupName,
    opts?: AnalyticsConfig,
  ): Promise<PlainMessage<AnalyticsViewResult>> {
    const inputFilters = opts?.filters ?? [];
    const columnMetaData = fillColumnMetaData(this.columnMetadata);
    const paginationSql = `LIMIT {limit:Int16} OFFSET {offset:Int16}`;

    const { result: coercedQueryParams, filterMapper } = coerceFilterValues(
      columnMetaData,
      inputFilters,
      this.baseFilters,
    );
    coercedQueryParams.limit = opts?.pagination?.limit ?? 30;
    coercedQueryParams.offset = opts?.pagination?.offset ?? 0;
    if (opts?.dateRange) {
      coercedQueryParams.startDate = Math.floor(new Date(opts.dateRange.start).getTime() / 1000);
      coercedQueryParams.endDate = Math.floor(new Date(opts.dateRange.end).getTime() / 1000);
    }

    const { havingSql, ...rest } = buildCoercedFilterSqlStatement(
      columnMetaData,
      coercedQueryParams,
      filterMapper,
      opts?.dateRange,
    );
    let { whereSql } = rest;

    // Important: This is the only place where we scope the data to a particular organization and graph.
    // We can only filter for data that is part of the JWT token otherwise a user could send us whatever they want.
    whereSql += ` AND SpanAttributes['wg.federated_graph.id'] = '${federatedGraphId}'`;
    whereSql += ` AND SpanAttributes['wg.organization.id'] = '${organizationId}'`;

    const [result, totalCount] = await Promise.all([
      this.getViewData(name, whereSql, havingSql, paginationSql, coercedQueryParams),
      this.getTotalCount(name, whereSql, havingSql, coercedQueryParams),
    ]);

    const hasColumn = (name: string) =>
      Array.isArray(result) && result.length > 0 && Object.keys(result[0]).includes(name);

    // We shall execute these only when we have desired results
    const [allOperationNames, allClientNames, allStatusMessages] = await Promise.all([
      this.getAllOperationNames(federatedGraphId, hasColumn('operationName')),
      this.getAllClients(federatedGraphId, hasColumn('clientName')),
      this.getAllHttpStatusCodes(federatedGraphId, hasColumn('httpStatusCode')),
    ]);

    const columnFilters = this.getFilters(name, allOperationNames, allClientNames, allStatusMessages);

    let pages = 0;
    if (totalCount > 0 && opts?.pagination?.limit) {
      pages = Math.ceil(totalCount / opts.pagination.limit);
    }

    /**
     * If no results, return empty rows but with default filters and columns
     */
    if (!Array.isArray(result) || result.length === 0) {
      const defaultColumns = buildColumnsFromNames(Object.keys(columnFilters), columnMetaData);
      const defaultFilters = Object.values(columnFilters).map(
        (f) =>
          ({
            columnName: f.columnName,
            title: f.title,
            options: f.options,
          } as PlainMessage<AnalyticsViewResultFilter>),
      );

      return {
        columns: defaultColumns,
        filters: defaultFilters,
        rows: [],
        pages,
      };
    }

    /**
     * If we have results, build the columns and filters based on the first row
     * Additional information to columns and filters is added from the columnMetaData and filters
     */

    const columns = buildAnalyticsViewColumns(result[0], columnMetaData);
    const filters = buildAnalyticsViewFilters(result[0], columnFilters);

    const rows: PlainMessage<AnalyticsViewRow>[] = result.map((row) => {
      const viewRow: Record<string, PlainMessage<AnalyticsViewRowValue>> = {};

      /**
       * JSON to protobuf conversion
       */
      for (const column in row) {
        const columnValue = row[column];
        if (typeof columnValue === 'string') {
          viewRow[column] = {
            kind: { case: 'stringValue', value: columnValue },
          };
        } else if (typeof columnValue === 'number') {
          viewRow[column] = {
            kind: { case: 'numberValue', value: columnValue },
          };
        } else if (typeof columnValue === 'boolean') {
          viewRow[column] = {
            kind: { case: 'boolValue', value: columnValue },
          };
        }
      }

      return {
        value: viewRow,
      };
    });

    return {
      filters,
      columns,
      rows,
      pages,
    };
  }
}
