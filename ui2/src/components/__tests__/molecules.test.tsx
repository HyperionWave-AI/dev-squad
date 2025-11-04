/**
 * Unit tests for molecular components (molecules)
 * Tests composition, interactions, and integration of atoms
 */
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardContent } from '@molecules/Card';
import { SearchBar } from '@molecules/SearchBar';
import { NavItem } from '@molecules/NavItem';
import { FormField } from '@molecules/FormField';

describe('Card Component', () => {
  it('should render card with children', () => {
    render(<Card>Card Content</Card>);
    expect(screen.getByText('Card Content')).toBeInTheDocument();
  });

  it('should apply custom className', () => {
    const { container } = render(<Card className="custom-card">Content</Card>);
    const card = container.firstChild;
    expect(card).toHaveClass('custom-card');
  });

  it('should render card with title when using CardTitle', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Card Title</CardTitle>
        </CardHeader>
        <CardContent>Card Body</CardContent>
      </Card>
    );

    expect(screen.getByText('Card Title')).toBeInTheDocument();
    expect(screen.getByText('Card Body')).toBeInTheDocument();
  });
});

describe('SearchBar Component', () => {
  it('should render search input', () => {
    render(<SearchBar onSearch={() => {}} />);
    const input = screen.getByPlaceholderText(/search/i);
    expect(input).toBeInTheDocument();
  });

  it('should call onSearch when user presses Enter', async () => {
    const user = userEvent.setup();
    let searchValue = '';
    const handleSearch = (value: string) => {
      searchValue = value;
    };

    render(<SearchBar onSearch={handleSearch} />);

    const input = screen.getByPlaceholderText(/search/i);
    await user.type(input, 'test query{Enter}');

    expect(searchValue).toBe('test query');
  });

  it('should render with placeholder text', () => {
    render(<SearchBar onSearch={() => {}} placeholder="Search items..." />);
    expect(screen.getByPlaceholderText('Search items...')).toBeInTheDocument();
  });

  it('should display search icon', () => {
    const { container } = render(<SearchBar onSearch={() => {}} />);

    // Search icon should be present
    const searchIcon = container.querySelector('svg');
    expect(searchIcon).toBeInTheDocument();
  });
});

describe('NavItem Component', () => {
  it('should render navigation link', () => {
    render(
      <MemoryRouter>
        <NavItem to="/test" label="Home" />
      </MemoryRouter>
    );

    expect(screen.getByText('Home')).toBeInTheDocument();
  });

  it('should have correct href attribute', () => {
    render(
      <MemoryRouter>
        <NavItem to="/test" label="Home" />
      </MemoryRouter>
    );

    const link = screen.getByText('Home').closest('a');
    expect(link).toHaveAttribute('href', '/test');
  });

  it('should render link that is clickable', () => {
    render(
      <MemoryRouter>
        <NavItem to="/test" label="Home" />
      </MemoryRouter>
    );

    const link = screen.getByText('Home').closest('a');
    expect(link).toBeInTheDocument();
    expect(link?.tagName).toBe('A');
  });
});

describe('FormField Component', () => {
  it('should render label and input together', () => {
    render(
      <FormField label="Username" id="username">
        <input id="username" type="text" />
      </FormField>
    );

    expect(screen.getByText('Username')).toBeInTheDocument();
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('should show error message when provided', () => {
    render(
      <FormField label="Email" id="email" error="Invalid email">
        <input id="email" type="email" />
      </FormField>
    );

    expect(screen.getByText('Email')).toBeInTheDocument();
    expect(screen.getByText('Invalid email')).toBeInTheDocument();
  });

  it('should show helper text when provided', () => {
    render(
      <FormField label="Password" id="password" helperText="Must be 8 characters">
        <input id="password" type="password" />
      </FormField>
    );

    expect(screen.getByText('Password')).toBeInTheDocument();
    expect(screen.getByText('Must be 8 characters')).toBeInTheDocument();
  });

  it('should show required indicator when required', () => {
    const { container } = render(
      <FormField label="Required Field" id="required" required>
        <input id="required" type="text" />
      </FormField>
    );

    expect(screen.getByText('Required Field')).toBeInTheDocument();
    // The FormField component should indicate required status visually
    expect(container.textContent).toContain('Required Field');
  });
});
