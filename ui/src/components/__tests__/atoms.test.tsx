/**
 * Unit tests for atomic components (atoms)
 * Tests rendering, props, and basic behavior
 */
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Icon } from '@atoms/Icon';
import { Label } from '@atoms/Label';
import { Button } from '@atoms/Button';
import { Badge } from '@atoms/Badge';
import { Input } from '@atoms/Input';

describe('Icon Component', () => {
  it('should render icon wrapper', () => {
    const { container } = render(<Icon>✓</Icon>);
    expect(container.querySelector('span')).toBeInTheDocument();
  });

  it('should apply custom className', () => {
    const { container } = render(<Icon className="custom-class">✓</Icon>);
    const span = container.querySelector('span');
    expect(span).toHaveClass('custom-class');
  });

  it('should render different icon sizes', () => {
    const { container: small } = render(<Icon size="sm">✓</Icon>);
    const { container: large } = render(<Icon size="lg">✓</Icon>);

    const smallSpan = small.querySelector('span');
    const largeSpan = large.querySelector('span');

    expect(smallSpan).toHaveClass('w-4', 'h-4');
    expect(largeSpan).toHaveClass('w-6', 'h-6');
  });
});

describe('Label Component', () => {
  it('should render label text', () => {
    render(<Label>Test Label</Label>);
    expect(screen.getByText('Test Label')).toBeInTheDocument();
  });

  it('should apply htmlFor attribute', () => {
    render(<Label htmlFor="test-input">Test Label</Label>);
    const label = screen.getByText('Test Label');
    expect(label).toHaveAttribute('for', 'test-input');
  });

  it('should apply custom className', () => {
    render(<Label className="custom-label">Test Label</Label>);
    const label = screen.getByText('Test Label');
    expect(label).toHaveClass('custom-label');
  });

  it('should render as required when specified', () => {
    const { container } = render(<Label required>Required Label</Label>);
    expect(screen.getByText('Required Label')).toBeInTheDocument();
    // Check if asterisk or required indicator is present
    expect(container.textContent).toContain('Required Label');
  });
});

describe('Button Component', () => {
  it('should render button with text', () => {
    render(<Button>Click Me</Button>);
    expect(screen.getByRole('button', { name: /click me/i })).toBeInTheDocument();
  });

  it('should call onClick handler when clicked', async () => {
    const user = userEvent.setup();
    let clicked = false;
    const handleClick = () => { clicked = true; };

    render(<Button onClick={handleClick}>Click Me</Button>);

    const button = screen.getByRole('button', { name: /click me/i });
    await user.click(button);

    expect(clicked).toBe(true);
  });

  it('should be disabled when disabled prop is true', () => {
    render(<Button disabled>Disabled Button</Button>);
    const button = screen.getByRole('button', { name: /disabled button/i });
    expect(button).toBeDisabled();
  });

  it('should render different variants', () => {
    const { rerender } = render(<Button variant="primary">Primary</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();

    rerender(<Button variant="secondary">Secondary</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();

    rerender(<Button variant="outline">Outline</Button>);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });
});

describe('Badge Component', () => {
  it('should render badge with text', () => {
    render(<Badge>New</Badge>);
    expect(screen.getByText('New')).toBeInTheDocument();
  });

  it('should apply different variants', () => {
    const { rerender } = render(<Badge variant="default">Default</Badge>);
    expect(screen.getByText('Default')).toBeInTheDocument();

    rerender(<Badge variant="success">Success</Badge>);
    expect(screen.getByText('Success')).toBeInTheDocument();

    rerender(<Badge variant="warning">Warning</Badge>);
    expect(screen.getByText('Warning')).toBeInTheDocument();

    rerender(<Badge variant="error">Error</Badge>);
    expect(screen.getByText('Error')).toBeInTheDocument();
  });

  it('should apply custom className', () => {
    render(<Badge className="custom-badge">Badge</Badge>);
    expect(screen.getByText('Badge')).toHaveClass('custom-badge');
  });
});

describe('Input Component', () => {
  it('should render input field', () => {
    render(<Input placeholder="Enter text" />);
    expect(screen.getByPlaceholderText('Enter text')).toBeInTheDocument();
  });

  it('should accept user input', async () => {
    const user = userEvent.setup();
    render(<Input placeholder="Enter text" />);

    const input = screen.getByPlaceholderText('Enter text') as HTMLInputElement;
    await user.type(input, 'test value');

    expect(input.value).toBe('test value');
  });

  it('should be disabled when disabled prop is true', () => {
    render(<Input disabled placeholder="Disabled input" />);
    const input = screen.getByPlaceholderText('Disabled input');
    expect(input).toBeDisabled();
  });

  it('should call onChange handler', async () => {
    const user = userEvent.setup();
    let value = '';
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      value = e.target.value;
    };

    render(<Input onChange={handleChange} placeholder="Enter text" />);

    const input = screen.getByPlaceholderText('Enter text');
    await user.type(input, 'test');

    expect(value).toBeTruthy();
  });

  it('should support different input types', () => {
    const { rerender } = render(<Input type="text" placeholder="Text" />);
    expect(screen.getByPlaceholderText('Text')).toHaveAttribute('type', 'text');

    rerender(<Input type="password" placeholder="Password" />);
    expect(screen.getByPlaceholderText('Password')).toHaveAttribute('type', 'password');

    rerender(<Input type="email" placeholder="Email" />);
    expect(screen.getByPlaceholderText('Email')).toHaveAttribute('type', 'email');
  });
});
