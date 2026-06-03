import { useEffect, useState } from 'react';
import { Box, Card, CardContent, Typography } from '@mui/material';
import { List, Datagrid, TextField } from 'react-admin';
import { TagsField } from '../TagsField';
import Leader from './Leader';
import DnsIcon from '@mui/icons-material/Dns';
import AssessmentIcon from '@mui/icons-material/Assessment';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import SpeedIcon from '@mui/icons-material/Speed';
import UpdateIcon from '@mui/icons-material/Update';
import ExecutionStatsChart from './ExecutionStatsChart';
import AnalyticsStats from './AnalyticsStats';
import CardWithIcon from './CardWithIcon';
import { apiUrl, httpClient } from '../dataProvider';

export interface NodeAnalytics {
    total_executions: number;
    successful_executions: number;
    failed_executions: number;
    average_duration_sec: number;
}

export interface AnalyticsData {
    total_jobs: number;
    total_executions: number;
    successful_executions: number;
    failed_executions: number;
    success_rate: number;
    failure_rate: number;
    average_duration_sec: number;
    min_duration_sec: number;
    max_duration_sec: number;
    duration_sample_count: number;
    last_execution_at?: string;
    executions_by_node?: Record<string, NodeAnalytics>;
}

const selectRowDisabled = () => false;

const fakeProps = {
    basePath: "/members",
    count: 10,
    hasCreate: false,
    hasEdit: false,
    hasList: true,
    hasShow: false,
    location: { pathname: "/", search: "", hash: "", state: undefined },
    match: { path: "/", url: "/", isExact: true, params: {} },
    options: {},
    permissions: null,
    resource: "members"
};

const formatInteger = (value?: number, fallback = "0") => (
    typeof value === 'number' ? value.toLocaleString() : fallback
);

const formatPercent = (value?: number) => (
    typeof value === 'number' ? `${(value * 100).toFixed(1)}%` : '0.0%'
);

const formatSeconds = (value?: number) => (
    typeof value === 'number' ? `${value.toFixed(2)}s` : '0.00s'
);

const Dashboard = () => {
    const [analytics, setAnalytics] = useState<AnalyticsData | null>(null);
    const [analyticsLoading, setAnalyticsLoading] = useState(true);
    const [analyticsError, setAnalyticsError] = useState<string | null>(null);

    useEffect(() => {
        let active = true;

        httpClient(`${apiUrl}/analytics`)
            .then(({ json }) => {
                if (active) {
                    setAnalytics(json as AnalyticsData);
                    setAnalyticsError(null);
                }
            })
            .catch((error) => {
                if (active) {
                    setAnalyticsError(error instanceof Error ? error.message : 'Unable to load analytics data');
                }
            })
            .finally(() => {
                if (active) {
                    setAnalyticsLoading(false);
                }
            });

        return () => {
            active = false;
        };
    }, []);

    return (
        <Box sx={{ p: { xs: 2, md: 3 } }}>
            {/* Header Section */}
            <Box sx={{ mb: 4 }}>
                <Typography
                    variant="h4"
                    component="h1"
                    sx={{
                        fontWeight: 700,
                        color: 'text.primary',
                        mb: 1,
                        fontSize: { xs: '1.75rem', md: '2.125rem' }
                    }}
                >
                    Dashboard
                </Typography>
                <Typography
                    variant="body1"
                    sx={{ color: 'text.secondary' }}
                >
                    Monitor your distributed job scheduler at a glance
                </Typography>
            </Box>

            {/* Stats Grid */}
            <Box
                sx={{
                    display: 'grid',
                    gridTemplateColumns: {
                        xs: '1fr',
                        sm: 'repeat(2, 1fr)',
                        md: 'repeat(3, 1fr)',
                        lg: 'repeat(6, 1fr)'
                    },
                    gap: { xs: 2, md: 3 },
                    mb: 4
                }}
            >
                <Leader value={window.DKRON_LEADER || "devel"} />
                <CardWithIcon
                    to="/jobs"
                    icon={UpdateIcon}
                    title="Total Jobs"
                    subtitle={formatInteger(analytics?.total_jobs, window.DKRON_TOTAL_JOBS || "0")}
                    color="#3182ce"
                />
                <CardWithIcon
                    to="/executions"
                    icon={AssessmentIcon}
                    title="Total Executions"
                    subtitle={formatInteger(analytics?.total_executions)}
                    color="#805ad5"
                />
                <CardWithIcon
                    to="/executions"
                    icon={CheckCircleIcon}
                    title="Success Rate"
                    subtitle={formatPercent(analytics?.success_rate)}
                    color="#38a169"
                />
                <CardWithIcon
                    to="/executions"
                    icon={ErrorIcon}
                    title="Failed Executions"
                    subtitle={formatInteger(analytics?.failed_executions, window.DKRON_FAILED_JOBS || "0")}
                    color="#e53e3e"
                />
                <CardWithIcon
                    to="/executions"
                    icon={SpeedIcon}
                    title="Avg Duration"
                    subtitle={formatSeconds(analytics?.average_duration_sec)}
                    color="#d69e2e"
                />
            </Box>

            <Box sx={{ mb: 4 }}>
                <AnalyticsStats
                    stats={analytics}
                    loading={analyticsLoading}
                    error={analyticsError}
                />
            </Box>

            {/* Execution Stats Chart */}
            <Box sx={{ mb: 4 }}>
                <ExecutionStatsChart />
            </Box>

            {/* Nodes Section */}
            <Card>
                <Box
                    sx={{
                        p: 3,
                        borderBottom: '1px solid',
                        borderColor: 'divider',
                        display: 'flex',
                        alignItems: 'center',
                        gap: 2
                    }}
                >
                    <Box
                        sx={{
                            width: 44,
                            height: 44,
                            borderRadius: 2,
                            background: 'linear-gradient(135deg, #1a365d 0%, #2c5282 100%)',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            color: 'white',
                            boxShadow: '0 4px 6px -1px rgba(26, 54, 93, 0.2)',
                        }}
                    >
                        <DnsIcon />
                    </Box>
                    <Box>
                        <Typography
                            variant="h6"
                            sx={{ fontWeight: 600, color: 'text.primary' }}
                        >
                            Cluster Nodes
                        </Typography>
                        <Typography
                            variant="body2"
                            sx={{ color: 'text.secondary' }}
                        >
                            Active members in your Dkron cluster
                        </Typography>
                    </Box>
                </Box>
                <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
                    <List {...fakeProps}>
                        <Datagrid
                            isRowSelectable={selectRowDisabled}
                            sx={{
                                '& .RaDatagrid-headerCell': {
                                    backgroundColor: '#f7fafc',
                                    fontWeight: 600,
                                    color: '#4a5568',
                                    fontSize: '0.75rem',
                                    textTransform: 'uppercase',
                                    letterSpacing: '0.05em',
                                },
                                '& .RaDatagrid-rowCell': {
                                    borderBottom: '1px solid #e2e8f0',
                                }
                            }}
                        >
                            <TextField source="Name" sortable={false} />
                            <TextField source="Addr" sortable={false} />
                            <TextField source="Port" sortable={false} />
                            <TextField label="Status" source="statusText" sortable={false} />
                            <TagsField />
                        </Datagrid>
                    </List>
                </CardContent>
            </Card>
        </Box>
    );
};

export default Dashboard;
