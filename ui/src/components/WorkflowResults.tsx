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
          <div className="animate-spin h-8 w-8 border-2 border-cb dark:border-db border-t-cprimary dark:border-t-dprimary rounded-full mx-auto mb-4"></div>
          <p className="text-ctext2 dark:text-dtext2">{translate('Running workflow...')}</p>
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="p-8 flex items-center justify-center">
        <div className="text-center">
          <p className="text-ctext2 dark:text-dtext2 text-lg">
            {translate('Click Run to execute the workflow')}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="p-4 space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm text-ctext2 dark:text-dtext2">
          <span className={`px-2 py-1 rounded text-sm font-medium ${data.success
            ? 'text-cprimary dark:text-dprimary'
            : 'bg-cerra text-cerr dark:bg-derra dark:text-derr'
            }`}>
            {data.success ? translate('Success') : translate('Failed')}
          </span>
        </div>
        <div className="text-xs text-ctext2 dark:text-dtext2 mr-4">
          {translate('Total Duration')}:
          <span className="ml-2">{data.queries.reduce((acc, q) => acc + q.duration, 0)}ms</span>
        </div>
      </div>

      {data.queries.map((query, index) => (
        <Card key={index}>
          <div className="space-y-3 p-4">
            <div className="flex items-start justify-between">
              <h3 className="text-sm font-medium text-ctext dark:text-dtext">
                {translate('Query')} {index + 1}
              </h3>
              <div className="flex items-center gap-2 text-xs text-ctext2 dark:text-dtext2">
                <span>{query.duration}ms</span>
                {query.error && (
                  <span className="px-2 py-1 bg-cerra text-cerr dark:bg-derra dark:text-derr rounded">
                    {translate('Error')}
                  </span>
                )}
              </div>
            </div>

            <pre className="text-xs bg-cbgs dark:bg-dbgs p-2 rounded border border-cb dark:border-db overflow-x-auto">
              <code>{query.sql}</code>
            </pre>

            {query.error && (
              <div className="p-2 bg-cerra dark:bg-derra border border-cerr dark:border-derr rounded">
                <p className="text-sm text-cerr dark:text-derr">
                  <strong>{translate('Error')}:</strong> {query.error}
                </p>
              </div>
            )}

            {query.result && query.result.length > 0 && (
              <div>
                <h4 className="text-sm font-medium mb-2 text-ctext dark:text-dtext">
                  {translate('Result')} ({query.result.length} {query.result.length === 1 ? translate('row') : translate('rows')})
                </h4>
                <div className="overflow-x-auto">
                  <Table>
                    <thead>
                      <tr>
                        {Object.keys(query.result[0]).map((column) => (
                          <th key={column} className="py-2 text-left text-xs font-medium text-ctext2 dark:text-dtext2 uppercase tracking-wider">
                            {column}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {query.result.map((row, rowIndex) => (
                        <tr key={rowIndex} className="border-t border-cb dark:border-db">
                          {Object.values(row).map((value, colIndex) => (
                            <td key={colIndex} className="py-2 text-sm">
                              {value === null ? (
                                <span className="text-ctext2 dark:text-dtext2 italic">null</span>
                              ) : (
                                <code className="text-ctext dark:text-dtext">{JSON.stringify(value, null, 2)}</code>
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
              <p className="text-sm text-ctext2 dark:text-dtext2">
                {translate('No rows returned')}
              </p>
            )}
          </div>
        </Card>
      ))}
    </div>
  )
}