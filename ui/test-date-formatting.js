/**
 * Test script to verify date formatting behavior in SessionList component
 * This tests the exact logic used in SessionList.tsx lines 158-169
 */

// Simulate the date-fns formatDistanceToNow function
function formatDistanceToNow(date, options = {}) {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  
  if (days > 0) return `${days} day${days > 1 ? 's' : ''} ago`;
  if (hours > 0) return `${hours} hour${hours > 1 ? 's' : ''} ago`;
  if (minutes > 0) return `${minutes} minute${minutes > 1 ? 's' : ''} ago`;
  return 'just now';
}

// Test the exact logic from SessionList.tsx
function testDateFormatting(timestamp, testName) {
  console.log(`\n=== Testing: ${testName} ===`);
  console.log(`Input: ${JSON.stringify(timestamp)}`);
  
  try {
    const date = typeof timestamp === 'string'
      ? new Date(timestamp)
      : timestamp;
    
    console.log(`Parsed date: ${date}`);
    console.log(`isNaN(date.getTime()): ${isNaN(date.getTime())}`);
    
    const result = isNaN(date.getTime())
      ? 'Invalid date'
      : formatDistanceToNow(date, { addSuffix: true });
    
    console.log(`Result: "${result}"`);
    return result;
  } catch (error) {
    console.log(`Caught error: ${error.message}`);
    return 'Invalid date';
  }
}

// Test cases covering various edge cases
console.log('Date Formatting Edge Case Tests');
console.log('================================');

// Valid cases
testDateFormatting('2024-01-15T10:30:00Z', 'Valid ISO string');
testDateFormatting(new Date(), 'Valid Date object');
testDateFormatting('2024-01-15T10:30:00.123Z', 'ISO string with milliseconds');

// Invalid cases that cause "Invalid Date"
testDateFormatting(null, 'null value');
testDateFormatting(undefined, 'undefined value');
testDateFormatting('', 'empty string');
testDateFormatting('invalid-date', 'invalid date string');
testDateFormatting('2024-13-45', 'invalid date components');
testDateFormatting(NaN, 'NaN value');
testDateFormatting({}, 'empty object');
testDateFormatting([], 'empty array');

// Edge cases
testDateFormatting('0', 'string "0"');
testDateFormatting(0, 'number 0 (Unix epoch)');
testDateFormatting('1970-01-01T00:00:00Z', 'Unix epoch as ISO string');

// Timezone issues
testDateFormatting('2024-01-15 10:30:00', 'Date without timezone');
testDateFormatting('2024-01-15T10:30:00', 'ISO without timezone suffix');

console.log('\n=== Summary ===');
console.log('The "Invalid Date" issue occurs when:');
console.log('1. timestamp is null or undefined');
console.log('2. timestamp is an empty string');
console.log('3. timestamp is a malformed date string');
console.log('4. timestamp is a non-date object type');
console.log('\nThe root cause in the application is that ChatSession objects');
console.log('have createdAt/updatedAt fields, but SessionList expects timestamp field.');