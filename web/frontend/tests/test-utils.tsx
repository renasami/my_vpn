import { render } from '@solidjs/testing-library';
import { Component, ComponentProps } from 'solid-js';
import { Router } from '@solidjs/router';
import { ThemeProvider, createTheme } from '@suid/material/styles';

// Create a theme for testing
const theme = createTheme();

// Create a test wrapper that provides necessary contexts
const TestWrapper = (props: { children: any }) => {
  return (
    <ThemeProvider theme={theme}>
      <Router>
        {props.children}
      </Router>
    </ThemeProvider>
  );
};

// Custom render function that wraps components with providers
export const renderWithProviders = (component: () => any) => {
  return render(() => <TestWrapper>{component()}</TestWrapper>);
};

export * from '@solidjs/testing-library';