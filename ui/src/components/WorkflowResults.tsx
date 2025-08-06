// SPDX-License-Identifier: MPL-2.0

import { translate } from '../lib/translate'
import { Card } from './tremor/Card'
import { Table } from './tremor/Table'

export interface WorkflowQueryResult {
  sql: string
  duration: number
  error?: string
  result?: Record<string, any>[]
}

export interface WorkflowResult {
  startTime: string
  success: boolean
  queries: WorkflowQueryResult[]
}

interface WorkflowResultsProps {
  data?: WorkflowResult
  loading?: boolean
}

export function WorkflowResults({ data, loading }: WorkflowResultsProps) {
  if (loading) {
    return (
      <div className="p-8 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin h-8 w-8 border-2 border-gray-300 border-t-blue-600 rounded-full mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">{translate('Running workflow...')}</p>
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="p-8 flex items-center justify-center">
        <div className="text-center">
          <p className="text-gray-600 dark:text-gray-400 text-lg">
            {translate('Click Run to execute the workflow')}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="p-4 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
          <span className={`px-2 py-1 rounded text-sm font-medium ${data.success
            ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
            : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
            }`}>
            {data.success ? translate('Success') : translate('Failed')}
          </span>
          <span>
            {translate('Started at')}: {new Date(data.startTime).toLocaleString()}
          </span>
        </div>
        <div className="text-xs text-gray-500 dark:text-gray-400 mr-4">
          {translate('Total Duration')}:
          <span className="ml-2">{data.queries.reduce((acc, q) => acc + q.duration, 0)}ms</span>
        </div>
      </div>

      {data.queries.map((query, index) => (
        <Card key={index}>
          <div className="space-y-3 p-4">
            <div className="flex items-start justify-between">
              <h3 className="text-sm font-medium text-gray-800 dark:text-gray-200">
                {translate('Query')} {index + 1}
              </h3>
              <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span>{query.duration}ms</span>
                {query.error && (
                  <span className="px-2 py-1 bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 rounded">
                    {translate('Error')}
                  </span>
                )}
              </div>
            </div>

            <pre className="text-xs bg-gray-50 dark:bg-gray-800 p-2 rounded border overflow-x-auto">
              <code>{query.sql}</code>
            </pre>

            {query.error && (
              <div className="p-2 bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800 rounded">
                <p className="text-sm text-red-800 dark:text-red-200">
                  <strong>{translate('Error')}:</strong> {query.error}
                </p>
              </div>
            )}

            {query.result && query.result.length > 0 && (
              <div>
                <h4 className="text-sm font-medium mb-2 text-gray-800 dark:text-gray-200">
                  {translate('Result')} ({query.result.length} {query.result.length === 1 ? translate('row') : translate('rows')})
                </h4>
                <div className="overflow-x-auto">
                  <Table>
                    <thead>
                      <tr>
                        {Object.keys(query.result[0]).map((column) => (
                          <th key={column} className="py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            {column}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {query.result.map((row, rowIndex) => (
                        <tr key={rowIndex} className="border-t border-gray-200 dark:border-gray-700">
                          {Object.values(row).map((value, colIndex) => (
                            <td key={colIndex} className="py-2 text-sm">
                              {value === null ? (
                                <span className="text-gray-400 italic">null</span>
                              ) : (
                                <code>{JSON.stringify(value, null, 2)}</code>
                              )}
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </Table>
                </div>
              </div>
            )}

            {query.result && query.result.length === 0 && (
              <p className="text-sm text-gray-600 dark:text-gray-400">
                {translate('No rows returned')}
              </p>
            )}
          </div>
        </Card>
      ))}
    </div>
  )
}