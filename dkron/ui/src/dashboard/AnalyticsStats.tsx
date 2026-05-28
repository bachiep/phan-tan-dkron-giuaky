import React, { useEffect, useState } from 'react';
import { Box, Card, Typography, CircularProgress } from '@mui/material';
import AssessmentIcon from '@mui/icons-material/Assessment';

const AnalyticsStats = () => {
    const [stats, setStats] = useState<any>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        fetch('/v1/analytics')
            .then(res => res.json())
            .then(data => {
                setStats(data);
                setLoading(false);
            })
            .catch(err => {
                console.error(err);
                setLoading(false);
            });
    }, []);

    if (loading) {
        return (
            <Card sx={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', p: 2 }}>
                <CircularProgress size={24} />
            </Card>
        );
    }

    if (!stats) {
        return null;
    }

    return (
        <Card sx={{ bgcolor: '#e3f2fd', height: '100%' }}>
            <Box sx={{ p: 2, display: 'flex', alignItems: 'center' }}>
                <AssessmentIcon sx={{ color: '#1565c0', fontSize: 40, mr: 2 }} />
                <Box>
                    <Typography variant="subtitle2" sx={{ color: '#1565c0', fontWeight: 'bold', textTransform: 'uppercase' }}>
                        Analytics API
                    </Typography>
                    <Typography variant="body2" color="textSecondary">
                        <strong>Rate:</strong> {(stats.success_rate * 100).toFixed(1)}% <br/>
                        <strong>Avg Run:</strong> {stats.average_duration_sec.toFixed(2)}s
                    </Typography>
                </Box>
            </Box>
        </Card>
    );
};

export default AnalyticsStats;
