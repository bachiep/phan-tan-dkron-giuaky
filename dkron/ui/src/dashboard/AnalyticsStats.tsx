import {
    Alert,
    Box,
    Card,
    CardContent,
    CircularProgress,
    Divider,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
    Typography,
} from '@mui/material';
import AssessmentIcon from '@mui/icons-material/Assessment';
import type { AnalyticsData, NodeAnalytics } from './Dashboard';

interface Props {
    stats: AnalyticsData | null;
    loading: boolean;
    error: string | null;
}

const formatInteger = (value?: number) => (
    typeof value === 'number' ? value.toLocaleString() : '0'
);

const formatPercent = (value?: number) => (
    typeof value === 'number' ? `${(value * 100).toFixed(1)}%` : '0.0%'
);

const formatSeconds = (value?: number) => (
    typeof value === 'number' ? `${value.toFixed(3)}s` : '0.000s'
);

const formatDateTime = (value?: string) => {
    if (!value) {
        return 'No execution recorded';
    }

    return new Date(value).toLocaleString();
};

const metricItems = (stats: AnalyticsData) => [
    { label: 'Successful executions', value: formatInteger(stats.successful_executions) },
    { label: 'Failed executions', value: formatInteger(stats.failed_executions) },
    { label: 'Failure rate', value: formatPercent(stats.failure_rate) },
    { label: 'Duration samples', value: formatInteger(stats.duration_sample_count) },
    { label: 'Minimum duration', value: formatSeconds(stats.min_duration_sec) },
    { label: 'Maximum duration', value: formatSeconds(stats.max_duration_sec) },
    { label: 'Last execution', value: formatDateTime(stats.last_execution_at) },
];

const nodeEntries = (nodes?: Record<string, NodeAnalytics>) => (
    Object.entries(nodes || {}).sort(([left], [right]) => left.localeCompare(right))
);

const AnalyticsStats = ({ stats, loading, error }: Props) => {
    return (
        <Card>
            <Box
                sx={{
                    p: 3,
                    borderBottom: '1px solid',
                    borderColor: 'divider',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 2,
                }}
            >
                <Box
                    sx={{
                        width: 44,
                        height: 44,
                        borderRadius: 2,
                        background: 'linear-gradient(135deg, #2b6cb0 0%, #805ad5 100%)',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        color: 'white',
                    }}
                >
                    <AssessmentIcon />
                </Box>
                <Box>
                    <Typography variant="h6" sx={{ fontWeight: 600, color: 'text.primary' }}>
                        Analytics API
                    </Typography>
                    <Typography variant="body2" sx={{ color: 'text.secondary' }}>
                        Runtime summary collected from Dkron execution history
                    </Typography>
                </Box>
            </Box>

            <CardContent>
                {loading && (
                    <Box sx={{ py: 4, display: 'flex', justifyContent: 'center' }}>
                        <CircularProgress size={28} />
                    </Box>
                )}

                {!loading && error && (
                    <Alert severity="warning">
                        Analytics data could not be loaded: {error}
                    </Alert>
                )}

                {!loading && !error && !stats && (
                    <Alert severity="info">
                        No analytics data is available yet.
                    </Alert>
                )}

                {!loading && !error && stats && (
                    <Box>
                        <Box
                            sx={{
                                display: 'grid',
                                gridTemplateColumns: {
                                    xs: '1fr',
                                    sm: 'repeat(2, 1fr)',
                                    md: 'repeat(4, 1fr)',
                                },
                                gap: 2,
                                mb: 3,
                            }}
                        >
                            {metricItems(stats).map((item) => (
                                <Box
                                    key={item.label}
                                    sx={{
                                        p: 2,
                                        border: '1px solid',
                                        borderColor: 'divider',
                                        borderRadius: 1,
                                        minHeight: 92,
                                    }}
                                >
                                    <Typography variant="caption" sx={{ color: 'text.secondary', fontWeight: 600 }}>
                                        {item.label}
                                    </Typography>
                                    <Typography variant="h6" sx={{ mt: 0.75, fontWeight: 700, lineHeight: 1.25 }}>
                                        {item.value}
                                    </Typography>
                                </Box>
                            ))}
                        </Box>

                        <Divider sx={{ mb: 2 }} />

                        <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 1.5 }}>
                            Executions by node
                        </Typography>
                        <Table size="small">
                            <TableHead>
                                <TableRow>
                                    <TableCell>Node</TableCell>
                                    <TableCell align="right">Total</TableCell>
                                    <TableCell align="right">Successful</TableCell>
                                    <TableCell align="right">Failed</TableCell>
                                    <TableCell align="right">Average duration</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {nodeEntries(stats.executions_by_node).map(([nodeName, nodeStats]) => (
                                    <TableRow key={nodeName}>
                                        <TableCell component="th" scope="row">
                                            {nodeName}
                                        </TableCell>
                                        <TableCell align="right">{formatInteger(nodeStats.total_executions)}</TableCell>
                                        <TableCell align="right">{formatInteger(nodeStats.successful_executions)}</TableCell>
                                        <TableCell align="right">{formatInteger(nodeStats.failed_executions)}</TableCell>
                                        <TableCell align="right">{formatSeconds(nodeStats.average_duration_sec)}</TableCell>
                                    </TableRow>
                                ))}
                                {nodeEntries(stats.executions_by_node).length === 0 && (
                                    <TableRow>
                                        <TableCell colSpan={5}>
                                            No node-level execution data is available.
                                        </TableCell>
                                    </TableRow>
                                )}
                            </TableBody>
                        </Table>
                    </Box>
                )}
            </CardContent>
        </Card>
    );
};

export default AnalyticsStats;
