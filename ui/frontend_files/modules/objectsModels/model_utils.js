/**
 * JavaScript equivalent of Go utility functions from model_utils.go
 */

/**
 * Converts an object to a BSON Map-like structure
 * Note: This is a simplified version since JavaScript doesn't have direct BSON handling
 */
export function toBsonM(data) {
  try {
    // In JavaScript, we can just return the object directly
    // This is a simplified equivalent since we don't need BSON conversion in frontend
    return JSON.parse(JSON.stringify(data));
  } catch (err) {
    console.error("Could not process data:", err);
    return null;
  }
}

/**
 * Converts a map to byte array (JSON string in JavaScript context)
 */
export function mapToByte(data) {
  try {
    return JSON.stringify(data);
  } catch (err) {
    console.error("Could not marshal data:", err);
    return null;
  }
}
